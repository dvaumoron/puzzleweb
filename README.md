# PuzzleWeb

<img src="https://github.com/dvaumoron/puzzleweb/raw/main/static/logo/puzzlelogo.jpg" width="100">

A library based on [Gin](https://github.com/gin-gonic/gin) to easily create server with static content, blog, wiki and forum.

Backed by microservices called with [gRPC](https://grpc.io/), those services definitions are :
- [puzzlesessionservice](https://github.com/dvaumoron/puzzlesessionservice) (this contract is also used for settings storage)
- [puzzleloginservice](https://github.com/dvaumoron/puzzleloginservice)
- [puzzlerightservice](https://github.com/dvaumoron/puzzlerightservice)
- [puzzleprofileservice](https://github.com/dvaumoron/puzzleprofileservice)
- [puzzleblogservice](https://github.com/dvaumoron/puzzleblogservice)
- [puzzlewikiservice](https://github.com/dvaumoron/puzzlewikiservice)
- [puzzleforumservice](https://github.com/dvaumoron/puzzleforumservice)
- [puzzlemarkdownservice](https://github.com/dvaumoron/puzzlemarkdownservice)

The [PuzzleTest](https://github.com/dvaumoron/puzzletest) project show how to use PuzzleWeb.