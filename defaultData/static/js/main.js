const go = new Go();
WebAssembly.instantiateStreaming(fetch("/static/puzzlefront.wasm"), go.importObject).then((result) => {
    go.run(result.instance);

    initWikiLink();
});
