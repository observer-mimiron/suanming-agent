package domain

import "testing"

func TestMonthlyStarIndex(t *testing.T) {
	tests := []struct {
		monthIndex int
		want       MonthlyStarIndex
	}{
		{monthIndex: 0, want: MonthlyStarIndex{Yuejie: 6, Tianyao: 11, Tianxing: 7, Yinsha: 0, Tianyue: 8, Tianwu: 3}},
		{monthIndex: 11, want: MonthlyStarIndex{Yuejie: 4, Tianyao: 10, Tianxing: 6, Yinsha: 2, Tianyue: 0, Tianwu: 9}},
	}
	for _, tt := range tests {
		if got := GetMonthlyStarIndex(tt.monthIndex); got != tt.want {
			t.Fatalf("GetMonthlyStarIndex(%d) = %+v, want %+v", tt.monthIndex, got, tt.want)
		}
	}
}

func TestDailyStarIndex(t *testing.T) {
	if got, want := GetDailyStarIndex(1, 0, 0), (DailyStarIndex{Santai: 2, Bazuo: 8, Enguang: 7, Tiangui: 1}); got != want {
		t.Fatalf("early zi = %+v, want %+v", got, want)
	}
	if got, want := GetDailyStarIndex(1, 0, 12), (DailyStarIndex{Santai: 3, Bazuo: 7, Enguang: 8, Tiangui: 2}); got != want {
		t.Fatalf("late zi = %+v, want %+v", got, want)
	}
}
