package statistic

import (
	"encoding/json"
	"fmt"
)

func convert(s Statistic) (StatisticDTO, error) {
	if s == nil {
		return StatisticDTO{}, fmt.Errorf("given Statistic is nil")
	}
	return StatisticDTO{
		Name:        s.Name(),
		Value:       s.Value(),
		Descriprion: s.Description(),
	}, nil
}

type StatisticDTO struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Descriprion string `json:"descriprion"`
	//Error       string `json:"error"`
}

func (o StatisticDTO) marshal() []byte {
	data, _ := json.Marshal(&o)
	return data
}

func (o StatisticDTO) convert() Statistic {
	return statisticProxy{dto: o}
}

type statisticProxy struct {
	dto StatisticDTO
}

func (s statisticProxy) Name() string {
	return s.dto.Name
}

func (s statisticProxy) Value() string {
	return s.dto.Value
}

func (s statisticProxy) Description() string {
	return s.dto.Descriprion
}

func unmarshal(data []byte) (StatisticDTO, error) {
	res := StatisticDTO{}
	err := json.Unmarshal(data, &res)
	if err != nil {
		return res, fmt.Errorf("cant parse StatisticDTO")
	}
	return res, nil
}
