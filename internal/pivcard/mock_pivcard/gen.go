package mock_pivcard

//go:generate -command mockgen go run github.com/golang/mock/mockgen
//go:generate mockgen -destination mock.go eagain.net/go/yubage/internal/pivcard Opener,Card
