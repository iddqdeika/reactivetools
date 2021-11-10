package statistic

// провайдер статистик
// интерфейс, который может вернуть разнообразные статистики.
// например, стандартная реализация CheckOrderProvider предоставляет данные об очереди заказов на проверку
type StatisticProvider interface {
	Statistics() ([]Statistic, error)
}

// статистика
// прредставляет единицу данных, отображающих статус/метрику некоего процесса/объекта
type Statistic interface {
	Name() string
	Value() string
	Description() string
}
