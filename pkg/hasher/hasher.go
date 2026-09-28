package hasher

type Hasher interface {
	Hash(input string) (string, error)
	Compare(input, hashedInput string) error
}
