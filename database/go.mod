module crawler/database

go 1.15

require github.com/lib/pq v1.9.0

require (
	github.com/e8kor/crawler/commons v0.0.0-20210104182532-faf38d3ccb2d
	github.com/openfaas/templates-sdk v0.0.0-20200723092016-0ebf61253625
)

replace github.com/e8kor/crawler/commons => ../commons
