// internal/delivery/telegram/services/signal/interface.go
package signal

type Service interface {
	Exec(params interface{}) (interface{}, error)
}
