module crawler/otodom/orchestrator

go 1.15

require (
    github.com/gocolly/colly/v2 v2.1.0
    github.com/openfaas/templates-sdk v0.0.0-20200723092016-0ebf61253625
)

replace (
	github.com/e8kor/crawler/commons => ../../commons
	github.com/e8kor/crawler/otodom/commons => ../commons
)