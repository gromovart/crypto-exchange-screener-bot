package counter

type Service interface {
	Exec(params interface{}) (interface{}, error)
}
