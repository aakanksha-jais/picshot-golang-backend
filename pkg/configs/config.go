package configs

type Config interface {
	Get(string) string
	GetOrDefault(string, string) string
}
