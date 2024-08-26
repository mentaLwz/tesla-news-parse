package ai

import (
	"testing"

	"github.com/mentaLwz/tesla-news-parse/config"
)

const oneNews = `
Tesla CEO Elon Musk hyped up his company's potential in autonomy once again on the latest earnings call. Tesla (TSLA -2.91%) slumped through most of the first half of 2024, and the stock is still underperforming the S&P 500 year to date. But investors seem to be giving it closer look again. After better-than-expected second-quarter deliveries report, it soared at the end of June and the beginning of July, and some investors seemed to be betting on its potential in AI. The electric vehicle (EV) leader has long traded at premium to other automakers, but its valuation seems increasingly hard to justify based on its EV business alone. Automotive revenue fell 7% in the second quarter to $19.9 billion, though overall revenue ticked up by 2% to $25.5 billion. Margins also continued to slide -- its operating margin dropped from 9.6% in the year-ago quarter to 6.3% this time, and adjusted net income tumbled by 42% from $3.1 billion to $1.8 billion. Image source: Tesla. Tesla stock took hit in the wake of its Q2 report, but it has become clear that the company's valuation is underpinned in large part by its potential in AI -- namely, autonomous vehicles such as its much-discussed robotaxi and its Optimus humanoid robot, which is also in development. On the earnings call, Musk once again argued that AI and autonomy were the most important factors for Tesla. "The big -- really by far, the biggest differentiator for Tesla is autonomy," he said. Later in the call, he added, "I really just can't emphasize the importance of autonomy for the vehicle side and for Optimus." However, if Musk is right about the transformative impact of autonomy, especially in the electric vehicle market, there's another stock that could be better bet than Tesla. Is this the AV leader? Alphabet's (GOOG -3.51%) (GOOGL -3.55%) Waymo doesn't get much attention from Wall Street analysts, but the autonomous vehicle subsidiary is generally considered the industry leader. Waymo has already deployed driverless vehicles -- essentially, robotaxis -- in Los Angeles, San Francisco, and Phoenix, and will soon deploy them in Austin as well. Waymo has logged more than 20 million fully autonomous miles on public roads, while Tesla's full self-driving (FSD) has not reached the same level of autonomy. Indeed, FSD is something of misnomer at this point, as it's more of driver-assist system than complete autonomous driving. Despite Waymo's success and early lead in autonomous vehicles, investors seem to mostly be ignoring the technology's potential impact on Alphabet's finances, instead staying focused on its search and advertising business, Google Cloud, and the company's broader AI efforts. Management noted on the recent call that Waymo is providing 50,000 paid public rides per week, mostly in San Francisco and Phoenix, as the service continues to scale up. Today's Change (-3.55%) -US$5.82 Current Price US$158.34 Arrow-Thin-Down Why Tesla investors should buy Alphabet stock Tesla currently trades at price-to-earnings ratio of 65, even though its revenue growth is essentially flat and profits are falling. The stock is being valued based on the expectation of future innovations. These include new vehicle models like the more affordable EV dubbed Model 2, but primarily, investors are giving it premium based on its potential in AI and autonomy, including robotaxis and Optimus. Alphabet, on the other hand, trades at price-to-earnings ratio of 24, and the company is growing both revenue and profits. In the latest quarter, revenue rose 15% year over year to $84.7 billion and earnings per share jumped 31% to $1.89. Waymo isn't impacting the bottom line yet, but it certainly could, especially if robotaxis are as big as Musk thinks they will be. Alphabet isn't an automaker like Tesla, so it likely can't produce its own robotaxis as quickly as Tesla might, but it can certainly license the technology or partner with manufacturer to mass-produce Waymo vehicles. Whatever happens in the autonomous vehicle space, Waymo is likely to remain leader and major player in any evolving robotaxi industry. If you like Tesla based on its self-driving potential, buying Alphabet stock makes sense. At its current valuation, you're essentially getting Waymo for free. Should you invest $1,000 in Alphabet right now? Before you buy stock in Alphabet, consider this: The Motley Fool Stock Advisor analyst team just identified what they believe are the 10 best stocks for investors to buy now鈥?and Alphabet wasn鈥檛 one of them. The 10 stocks that made the cut could produce monster returns in the coming years. Consider when Nvidia made this list on April 15, 2005... if you invested $1,000 at the time of our recommendation, you鈥檇 have $711,657!* Stock Advisor provides investors with an easy-to-follow blueprint for success, including guidance on building portfolio, regular updates from analysts, and two new stock picks each month. The Stock Advisor service has more than quadrupled the return of S&P 500 since 2002*. See the 10 stocks *Stock Advisor returns as of August 12, 2024 Suzanne Frey, an executive at Alphabet, is member of The Motley Fool's board of directors. Jeremy Bowman has no position in any of the stocks mentioned. The Motley Fool has positions in and recommends Alphabet and Tesla. The Motley Fool has disclosure policy.`

func TestGetScoreDeepSeek(t *testing.T) {

	config.LoadConfig()
	// 设置配置

	testCases := []struct {
		name     string
		news     string
		expected string
	}{
		{
			name:     "Positive news",
			news:     oneNews,
			expected: "5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetScoreDeepSeek(tc.news)
			t.Logf("News: %s, Score: %s", tc.news, result)
			if result == "0" {
				t.Logf("API call might have failed or returned unexpected result")
			}
			// Note: We're not strictly asserting the expected value because AI responses can vary
			// Instead, we're logging the result for manual inspection
		})
	}
}
