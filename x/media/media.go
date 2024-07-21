package media

type AudioPlayer interface {
	Play(data []byte) error
	Stop() error
}
