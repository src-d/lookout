package api

//go:generate protoc --gofast_out=plugins=grpc:. api.proto

type Service interface {
	GetChanges(*ChangesRequest) (ChangeScanner, error)
}

type ChangeScanner interface {
	Next() bool
	Err() error
	Change() *Change
	Close() error
}
