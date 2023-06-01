# PuzzleWeb

<img src="https://github.com/dvaumoron/puzzleweb/raw/main/defaultData/static/images/puzzlelogo.jpg" width="100">

A library based on [Gin](https://gin-gonic.com/) to easily create server with static content, blog, wiki and forum.

The projects [PuzzleFrame](https://github.com/dvaumoron/puzzleframe) (with [frame.yaml](https://github.com/dvaumoron/puzzleframe/blob/main/frame.yaml)) and [PuzzleTest](https://github.com/dvaumoron/puzzletest) (with [puzzletest.go](https://github.com/dvaumoron/puzzletest/blob/main/puzzletest.go)) show how to use PuzzleWeb.

The main server is backed by microservices called with [gRPC](https://grpc.io/), those services definitions are :
- [puzzlesessionservice](https://github.com/dvaumoron/puzzlesessionservice) (this contract is also used for settings storage)
- [puzzletemplateservice](https://github.com/dvaumoron/puzzletemplateservice)
- [puzzlepassstrengthservice](https://github.com/dvaumoron/puzzlepassstrengthservice)
- [puzzlesaltservice](https://github.com/dvaumoron/puzzlesaltservice)
- [puzzleloginservice](https://github.com/dvaumoron/puzzleloginservice)
- [puzzlerightservice](https://github.com/dvaumoron/puzzlerightservice)
- [puzzleprofileservice](https://github.com/dvaumoron/puzzleprofileservice)

And optionnally (with some kind of page added) :
- [puzzleforumservice](https://github.com/dvaumoron/puzzleforumservice)
- [puzzlemarkdownservice](https://github.com/dvaumoron/puzzlemarkdownservice)
- [puzzleblogservice](https://github.com/dvaumoron/puzzleblogservice)
- [puzzlewikiservice](https://github.com/dvaumoron/puzzlewikiservice)
- [puzzlewidgetservice](https://github.com/dvaumoron/puzzlewidgetservice), which is a way to add your custom dynamic page in a decoupled way

List of helper projects :
- [puzzlegrpcserver](https://github.com/dvaumoron/puzzlegrpcserver)
- [puzzlegrpcclient](https://github.com/dvaumoron/puzzlegrpcclient)
- [puzzledbclient](https://github.com/dvaumoron/puzzledbclient)
- [puzzlemongoclient](https://github.com/dvaumoron/puzzlemongoclient)
- [puzzleredisclient](https://github.com/dvaumoron/puzzleredisclient)
- [puzzlesaltclient](https://github.com/dvaumoron/puzzlesaltclient)
- [puzzletelemetry](https://github.com/dvaumoron/puzzletelemetry)
- [puzzlewidgetserver](https://github.com/dvaumoron/puzzlewidgetserver)
