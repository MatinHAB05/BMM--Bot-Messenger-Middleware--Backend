package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	service_contract "messenger-backend/internal/application/contract"
	"messenger-backend/internal/domain/exception"
	"messenger-backend/internal/domain/otp"
	repository_contract "messenger-backend/internal/domain/repository"
	redisApt "messenger-backend/internal/infrastructure/redis"
	"messenger-backend/pkg/logger"
)

// otpService implements service_contract.OTPService. It is deliberately
// thin: every behavior that differs by OTP type -- key formatting, TTL,
// code format, payload shape -- lives in the otp.Strategy registered for
// that type, so adding a new OTP type never touches this file.
type otpService struct {
	repo              repository_contract.OTPRepository
	trx               redisApt.TrxManager
	strategies        map[otp.Type]otp.Strategy
	emailDelivery     service_contract.EmailService
	appEnv            string
	debugEmailAddress string
	log               logger.Logger
}

// NewOTPService builds the service around whichever strategies the
// caller registers. Passing strategies in at construction -- rather
// than this file importing and hardcoding
// otp.NewPhoneStrategy/NewEmailStrategy/NewLinkStrategy itself -- keeps
// the Open/Closed promise real: a deployment that wants a fourth OTP
// type registers one more Strategy here and changes nothing else in the
// module.
func NewOTPService(
	repo repository_contract.OTPRepository,
	trx redisApt.TrxManager,
	emailDelivery service_contract.EmailService,
	appEnv string,
	debugEmailAddress string,
	log logger.Logger,

	strategies ...otp.Strategy,
) (service_contract.OTPService, error) {
	if len(strategies) == 0 {
		return nil, fmt.Errorf("otp service: at least one strategy must be registered")
	}

	reg := make(map[otp.Type]otp.Strategy, len(strategies))
	for _, s := range strategies {
		if _, dup := reg[s.Type()]; dup {
			return nil, fmt.Errorf("otp service: duplicate strategy registered for type %q", s.Type())
		}
		reg[s.Type()] = s
	}

	return &otpService{
		repo:          repo,
		trx:           trx,
		emailDelivery: emailDelivery,
		strategies:    reg,
		debugEmailAddress :  debugEmailAddress,
		appEnv:        appEnv,
		log:           log.With(logger.String("component", "otp_service")),
	}, nil
}

func (s *otpService) strategyFor(otpType otp.Type) (otp.Strategy, error) {
	strat, ok := s.strategies[otpType]
	if !ok {
		return nil, fmt.Errorf("%w: %q", exception.ErrOTPUnsupportedType, otpType)
	}
	return strat, nil
}

// SendOTP generates a code via otpType's Strategy, wraps it in that
// strategy's Payload together with any caller-supplied metadata,
// serializes the payload to JSON, and persists it under the strategy's
// key with the strategy's default TTL.
func (s *otpService) SendOTP(ctx context.Context, identifier string, otpType otp.Type, metadata map[string]any) (*service_contract.SendOTPResult, error) {
	strat, err := s.strategyFor(otpType)
	if err != nil {
		return nil, err
	}

	code, err := strat.GenerateCode()
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	payload, err := strat.NewPayload(code, metadata)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, fmt.Errorf("marshal otp payload: %w", err))
	}

	ttl := strat.DefaultTTL()
	key := strat.Key(identifier, code)
	if err := s.repo.Save(ctx, key, raw, ttl); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	result := &service_contract.SendOTPResult{
		Message:         "otp sent",
		ExpiresInSecond: int(ttl.Seconds()),
	}

	switch otpType {
	case otp.TypeEmail:
		// for now just try send email [just try]
		dest := identifier
		if  s.appEnv == "development" { //TODO
			dest = s.debugEmailAddress
		}
		ok, err := s.emailDelivery.SendOTPEmail(ctx, service_contract.SendOTPEmailRequest{
			To:  []string{dest},
			OTP: code,
			TTL: timeToPrettyFormat(ttl),
		})
		if err != nil {
			s.log.Error(err, "email otp deliviery has failed[for now just debuging]", logger.Bool("ok", ok != nil && *ok))
		}
	case otp.TypePhone:
		s.log.Info("phone otp deliviery not implemented yet")
	}

	// Mirrors the existing AuthService.SendOTP convention: no delivery
	// provider is wired in yet, so outside production the code is
	// logged AND returned so the flow is testable end-to-end; in
	// production it's only ever logged.
	if s.appEnv == "production" {
		s.log.Info("otp generated", logger.String("type", string(otpType)), logger.String("identifier", identifier))
	} else {
		result.Code = code
		s.log.Info("otp generated (non-production build, code included in response)",
			logger.String("type", string(otpType)), logger.String("identifier", identifier))
	}

	return result, nil
}

// VerifyOTP looks up the payload stored for (identifier, otpType, code)
// via the strategy's key formatting, then compares the stored code
// against the submitted one using a constant-time comparison -- OTP
// codes are short-lived secrets, so there's no reason to leak timing
// information about a partial match.
func (s *otpService) VerifyOTP(ctx context.Context, identifier string, otpType otp.Type, code string) (otp.Payload, error) {
	strat, err := s.strategyFor(otpType)
	if err != nil {
		return nil, err
	}

	key := strat.Key(identifier, code)
	raw, err := s.repo.Get(ctx, key)
	if err != nil {
		if errors.Is(err, exception.ErrOTPNotFound) {
			return nil, exception.ErrOTPInvalid
		}
		return nil, exception.Wrap(exception.ErrInternal, err)
	}

	payload := strat.EmptyPayload()
	if err := json.Unmarshal(raw, payload); err != nil {
		return nil, exception.Wrap(exception.ErrInternal, fmt.Errorf("unmarshal otp payload: %w", err))
	}

	if subtle.ConstantTimeCompare([]byte(payload.Code()), []byte(code)) != 1 {
		return nil, exception.ErrOTPInvalid
	}

	return payload, nil
}

func (s *otpService) IsVerified(ctx context.Context, identifier string, otpType otp.Type) (*bool, error) {
	strat, err := s.strategyFor(otpType)
	if err != nil {
		return nil, err
	}

	// key := strat.Key(identifier, code)
	key := strat.KeyWithID(identifier)
	ok, err := s.repo.IsVerified(ctx, key)
	if err != nil {
		return nil, exception.Wrap(exception.ErrInternal, err)
	}
	return ok, err
}

// InvalidateIsVerified atomically checks-and-clears the "verified" flag: the
// flag key is WATCHed, so two concurrent callers can't both consume it.
// Returns true only if this call actually consumed a set flag.
func (s *otpService) InvalidateIsVerified(ctx context.Context, identifier string, otpType otp.Type) (*bool, error) {
	strat, err := s.strategyFor(otpType)
	if err != nil {
		return nil, err
	}

	key := strat.KeyWithID(identifier)
	var was bool

	err = s.trx.WithWatch(ctx, []string{s.repo.VerifiedKey(key)},
		func(trxCtx context.Context) error {
			ok, err := s.repo.IsVerified(trxCtx, key)
			if err != nil {
				return exception.Wrap(exception.ErrInternal, err)
			}
			was = ok != nil && *ok
			return nil
		},
		func(trxCtx context.Context) error {
			if !was {
				return nil
			}
			if err := s.repo.RemoveVerified(trxCtx, key); err != nil {
				return exception.Wrap(exception.ErrInternal, err)
			}
			return nil
		},
	)
	if err != nil {
		return nil, s.trxErr(err)
	}

	return &was, nil
}

// InvalidateOTP does NOT run automatically after a successful
// VerifyOTP -- callers that want single-use semantics call this
// themselves right after verifying, same as the existing AuthService
// does today. Keeping it a separate step lets a caller verify without
// necessarily burning the code (e.g. to preview a link's payload before
// consuming it).
func (s *otpService) InvalidateOTP(ctx context.Context, identifier string, otpType otp.Type, code string) error {
	strat, err := s.strategyFor(otpType)
	if err != nil {
		return err
	}

	key := strat.Key(identifier, code)
	if err := s.repo.Delete(ctx, key); err != nil {
		if errors.Is(err, exception.ErrOTPNotFound) {
			return exception.ErrOTPNotFound
		}
		return exception.Wrap(exception.ErrInternal, err)
	}
	return nil
}

// InvalidateOTPAndSetVerified atomically burns the OTP and raises the
// "verified" flag. The OTP key is WATCHed: if it disappears (or changes)
// between the existence check and EXEC, nothing is written and the check is
// retried, so an OTP can only ever be exchanged for a verified flag once.
func (s *otpService) InvalidateOTPAndSetVerified(ctx context.Context, identifier string, otpType otp.Type, code string) error {
	strat, err := s.strategyFor(otpType)
	if err != nil {
		return err
	}

	otpKey := strat.Key(identifier, code)
	verKey := strat.KeyWithID(identifier)

	err = s.trx.WithWatch(ctx, []string{otpKey},
		func(trxCtx context.Context) error {
			exists, err := s.repo.Exists(trxCtx, otpKey)
			if err != nil {
				return exception.Wrap(exception.ErrInternal, err)
			}
			if !exists {
				return exception.ErrOTPNotFound
			}
			return nil
		},
		func(trxCtx context.Context) error {
			if err := s.repo.Remove(trxCtx, otpKey); err != nil {
				return exception.Wrap(exception.ErrInternal, err)
			}
			if err := s.repo.SetVerfied(trxCtx, verKey, strat.DefaultTTL()); err != nil {
				return exception.Wrap(exception.ErrInternal, err)
			}
			return nil
		},
	)
	if err != nil {
		return s.trxErr(err)
	}
	return nil
}

// GetAndInvalidateOTP verifies the code and burns it in one atomic step:
// the OTP key is WATCHed, so two concurrent submissions of the same code can't
// both succeed (the loser retries, finds nothing, and gets ErrOTPInvalid).
func (s *otpService) GetAndInvalidateOTP(ctx context.Context, identifier string, otpType otp.Type, code string) (otp.Payload, error) {
	strat, err := s.strategyFor(otpType)
	if err != nil {
		return nil, err
	}

	key := strat.Key(identifier, code)
	var payload otp.Payload

	err = s.trx.WithWatch(ctx, []string{key},
		func(trxCtx context.Context) error {
			p, err := s.VerifyOTP(trxCtx, identifier, otpType, code)
			if err != nil {
				return err
			}
			payload = p
			return nil
		},
		func(trxCtx context.Context) error {
			if err := s.repo.Remove(trxCtx, key); err != nil {
				return exception.Wrap(exception.ErrInternal, err)
			}
			return nil
		},
	)
	if err != nil {
		return nil, s.trxErr(err)
	}
	return payload, nil
}

// trxErr maps "watch retries exhausted" to an internal error; every other
// error already carries the right domain/internal wrapping.
func (s *otpService) trxErr(err error) error {
	if errors.Is(err, redisApt.ErrTrxConflict) {
		return exception.Wrap(exception.ErrInternal, err)
	}
	return err
}

func (s *otpService) TTL(ctx context.Context, identifier string, otpType otp.Type, code string) (time.Duration, error) {
	strat, err := s.strategyFor(otpType)
	if err != nil {
		return 0, err
	}

	key := strat.Key(identifier, code)
	ttl, err := s.repo.TTL(ctx, key)
	if err != nil {
		if errors.Is(err, exception.ErrOTPNotFound) {
			return 0, exception.ErrOTPNotFound
		}
		return 0, exception.Wrap(exception.ErrInternal, err)
	}
	return ttl, nil
}

func (s *otpService) RegisteredTypes() []otp.Type {
	types := make([]otp.Type, 0, len(s.strategies))
	for t := range s.strategies {
		types = append(types, t)
	}
	return types
}
