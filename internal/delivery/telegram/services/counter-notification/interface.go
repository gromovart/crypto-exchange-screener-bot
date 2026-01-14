package counternotification

type Service interface {
	Exec(params interface{}) (interface{}, error)
}
