# PuzzleWeb

<img src="https://github.com/dvaumoron/puzzleweb/raw/main/defaultData/static/images/puzzlelogo.jpg" width="100">

This library is intended to easily create a microservice backed server allowing to include static content, blog, wiki, forum and custom "widget" with role based right management, user profile, user settings, and [i18n](https://www.w3.org/International/questions/qa-i18n.en#i18n).

## License

All of the project in the Puzzle ecosystem are released under the Apache 2.0 license. See [LICENSE](LICENSE).

## Getting started

The project [PuzzleWeaver](https://github.com/dvaumoron/puzzleweaver) allows to use PuzzleWeb features with a single binary (it is a modular monolith done with [ServiceWeaver](https://serviceweaver.dev/) and configured with [puzzleweaver.toml](https://github.com/dvaumoron/puzzleweaver/blob/main/puzzleweaver.toml)). Once installed, you can run it with the command :

    weaver single deploy puzzleweaver.toml

Other projects based on PuzzleWeb are [PuzzleFrame](https://github.com/dvaumoron/puzzleframe) (configured with [frame.yaml](https://github.com/dvaumoron/puzzleframe/blob/main/frame.yaml)) and [PuzzleTest](https://github.com/dvaumoron/puzzletest) (initialized in [puzzletest.go](https://github.com/dvaumoron/puzzletest/blob/main/puzzletest.go)). PuzzleFrame and PuzzleTest needs additionnal configuration to be provided with environment variable (or a .env file in the working directory, see [this empty exemple](defaultData/.env)) and the backing services should be handled separately.

See [this folder](https://github.com/dvaumoron/puzzletest/tree/main/deploy/conf/helm) for an example of [Helm chart](https://helm.sh).

See [API Documentation](https://pkg.go.dev/github.com/dvaumoron/puzzleweb) for detailed package descriptions.

## Technical overview

The main server use [Gin](https://gin-gonic.com/) and is backed by microservices called with [gRPC](https://grpc.io/), those services definitions (and list of proposed implementations) are :

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
    - [puzzlerightserver](https://github.com/dvaumoron/puzzlerightserver) (use Rego from [Open Policy Agent](https://www.openpolicyagent.org/))
    - [puzzlecachedrightserver](https://github.com/dvaumoron/puzzlecachedrightserver)
7. [puzzleprofileservice](https://github.com/dvaumoron/puzzleprofileservice)
    - [puzzleprofileserver](https://github.com/dvaumoron/puzzleprofileserver)

And optionnally (with some kind of page added) :

1. [puzzleforumservice](https://github.com/dvaumoron/puzzleforumservice)
    - [puzzleforumserver](https://github.com/dvaumoron/puzzleforumserver)
2. [puzzlemarkdownservice](https://github.com/dvaumoron/puzzlemarkdownservice)
    - [puzzlemarkdownserver](https://github.com/dvaumoron/puzzlemarkdownserver)
3. [puzzleblogservice](https://github.com/dvaumoron/puzzleblogservice)
    - [puzzleblogserver](https://github.com/dvaumoron/puzzleblogserver)
4. [puzzlewikiservice](https://github.com/dvaumoron/puzzlewikiservice)
    - [puzzlewikiserver](https://github.com/dvaumoron/puzzlewikiserver)
5. [puzzlewidgetservice](https://github.com/dvaumoron/puzzlewidgetservice), which is a way to add your custom dynamic page in a decoupled way
    - [puzzlegalleryserver](https://github.com/dvaumoron/puzzlegalleryserver) : Image gallery

List of side projects:

- [puzzlefront](https://github.com/dvaumoron/puzzlefront) : [WebAssembly](https://webassembly.org/) project containing the majority of browser side interaction.
- [puzzletools](https://github.com/dvaumoron/puzzletools) : [Cobra](https://cobra.dev/) based utility CLI.

List of helper projects :

- [puzzlegrpcserver](https://github.com/dvaumoron/puzzlegrpcserver)
- [puzzlegrpcclient](https://github.com/dvaumoron/puzzlegrpcclient)
- [puzzledbclient](https://github.com/dvaumoron/puzzledbclient) (use [gorm](https://gorm.io/))
- [puzzlemongoclient](https://github.com/dvaumoron/puzzlemongoclient)
- [puzzleredisclient](https://github.com/dvaumoron/puzzleredisclient)
- [puzzletelemetry](https://github.com/dvaumoron/puzzletelemetry) (use [OpenTelemetry](https://opentelemetry.io/) and [Zap](https://pkg.go.dev/go.uber.org/zap))
- [puzzlesaltclient](https://github.com/dvaumoron/puzzlesaltclient) (use [x/crypto](https://pkg.go.dev/golang.org/x/crypto))
- [puzzlewidgetserver](https://github.com/dvaumoron/puzzlewidgetserver)
