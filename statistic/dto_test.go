package statistic

import "testing"

var n, v, d, e = "name", "val", "desc", "err"

func TestDto(t *testing.T) {
	testStatisticDTO(t, n, v, d, e)
}

func testStatisticDTO(t *testing.T, n string, v string, d string, e string) {
	dto := StatisticDTO{
		Name:        n,
		Value:       v,
		Descriprion: d,
		Error:       e,
	}
	data := dto.marshal()
	res, err := unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if res.Name != dto.Name ||
		res.Value != dto.Value ||
		res.Descriprion != dto.Descriprion ||
		res.Error != dto.Error {
		t.Fatal("incorrect StatisticDTO data after marshal-unmarshal")
	}
}
