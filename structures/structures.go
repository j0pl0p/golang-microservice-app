package structures

// ExpressionDataJSON жсончик для получения данных о выражении
type ExpressionDataJSON struct {
	Exp string `json:"expression"`
}

// IdReceiveJSON жсончик для получения айдишника
type IdReceiveJSON struct {
	Id string `json:"id"`
}

// CalcDurationsJSON жсончик для данных о длительности каждого из операторов
type CalcDurationsJSON struct {
	Plus  int `json:"plus"`
	Minus int `json:"minus"`
	Mul   int `json:"mul"`
	Div   int `json:"div"`
}

// Expression Структура выражения
type Expression struct {
	Exp    string
	Id     string
	Status string
	Result float32
}
