package do

// DailyRecommendation 表示单张每日推荐领域对象。
type DailyRecommendation struct {
	Date     string
	Image    Image
	Position int
	Cycle    int64
}

// DailyRecommendationList 表示某自然日的全站每日推荐结果。
type DailyRecommendationList struct {
	Date   string
	Images []Image
}
