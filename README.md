# PuzzleWeb

<img src="https://github.com/dvaumoron/puzzleweb/raw/main/defaultData/static/images/puzzlelogo.jpg" width="100">

A library based on [Gin](https://gin-gonic.com/) to easily create a microservice backed server allowing to include static content, blog, wiki, forum and custom "widget" with role based right management, user profile, user settings, and [i18n](https://www.w3.org/International/questions/qa-i18n.en#i18n).

## License

All of the project in the Puzzle ecosystem are released under the Apache 2.0 license. See [LICENSE](LICENSE)

## Getting started

The projects [PuzzleFrame](https://github.com/dvaumoron/puzzleframe) (configured with [frame.yaml](https://github.com/dvaumoron/puzzleframe/blob/main/frame.yaml)) and [PuzzleTest](https://github.com/dvaumoron/puzzletest) (initialized in [puzzletest.go](https://github.com/dvaumoron/puzzletest/blob/main/puzzletest.go)) show how to use PuzzleWeb. In both case, additionnal configuration should be provided with environment variable (or a .env file in the working directory, see [this empty exemple](defaultData/.env)).

See [API Documentation](https://pkg.go.dev/github.com/dvaumoron/puzzleweb) for detailed package descriptions.

## Technical overview

The main server is backed by microservices called with [gRPC](https://grpc.io/), those services definitions (and list of proposed implementations) are :
1. [puzzlesessionservice](https://github.com/dvaumoron/puzzlesessionservice) (this contract is also used for settings storage)
    - [puzzlesessionserver](https://github.com/dvaumoron/puzzlesessionserver)
    - [puzzlesettingsserver](https://github.com/dvaumoron/puzzlesettingsserver)
2. [puzzletemplateservice](https://github.com/dvaumoron/puzzletemplateservice)
    - [puzzlegotemplateserver](https://github.com/dvaumoron/puzzlegotemplateserver)
    - [puzzleindentlangserver](https://github.com/dvaumoron/puzzleindentlangserver)
3. [puzzlepassstrengthservice](https://github.com/dvaumoron/puzzlepassstrengthservice)
    - [puzzlepassstrengthserver](https://github.com/dvaumoron/puzzlepassstrengthserver)
4. [puzzlesaltservice](https://github.com/dvaumoron/puzzlesaltservice)
    - [puzzlesaltserver](https://github.com/dvaumoron/puzzlesaltserver)
5. [puzzleloginservice](https://github.com/dvaumoron/puzzleloginservice)
    - [puzzleloginserver](https://github.com/dvaumoron/puzzleloginserver)
6. [puzzlerightservice](https://github.com/dvaumoron/puzzlerightservice)
    - [puzzlerightserver](https://github.com/dvaumoron/puzzlerightserver)
    - [puzzlecachedrightserver](https://github.com/dvaumoron/puzzlecachedrightserver)
7. [puzzleprofileservice](https://github.com/dvaumoron/puzzleprofileservice)
    - [puzzleprofileserver](https://github.com/dvaumoron/puzzleprofileserver)

And optionnally (with some kind of page added) :
8. [puzzleforumservice](https://github.com/dvaumoron/puzzleforumservice)
    - [puzzleforumserver](https://github.com/dvaumoron/puzzleforumserver)
9. [puzzlemarkdownservice](https://github.com/dvaumoron/puzzlemarkdownservice)
    - [puzzlemarkdownserver](https://github.com/dvaumoron/puzzlemarkdownserver)
10. [puzzleblogservice](https://github.com/dvaumoron/puzzleblogservice)
    - [puzzleblogserver](https://github.com/dvaumoron/puzzleblogserver)
11. [puzzlewikiservice](https://github.com/dvaumoron/puzzlewikiservice)
    - [puzzlewikiserver](https://github.com/dvaumoron/puzzlewikiserver)
12. [puzzlewidgetservice](https://github.com/dvaumoron/puzzlewidgetservice), which is a way to add your custom dynamic page in a decoupled way
    - [puzzlegalleryserver](https://github.com/dvaumoron/puzzlegalleryserver) : Image gallery

List of side projects:
- [puzzlefront](https://github.com/dvaumoron/puzzlefront) : [WebAssembly](https://webassembly.org/) project containing the majority of browser side interaction.
- [puzzletools](https://github.com/dvaumoron/puzzletools) : [Cobra](https://cobra.dev/) based utility CLI.

List of helper projects :
- [puzzlegrpcserver](https://github.com/dvaumoron/puzzlegrpcserver)
- [puzzlegrpcclient](https://github.com/dvaumoron/puzzlegrpcclient)
- [puzzledbclient](https://github.com/dvaumoron/puzzledbclient)
- [puzzlemongoclient](https://github.com/dvaumoron/puzzlemongoclient)
- [puzzleredisclient](https://github.com/dvaumoron/puzzleredisclient)
- [puzzlesaltclient](https://github.com/dvaumoron/puzzlesaltclient)
- [puzzletelemetry](https://github.com/dvaumoron/puzzletelemetry)
- [puzzlewidgetserver](https://github.com/dvaumoron/puzzlewidgetserver)
