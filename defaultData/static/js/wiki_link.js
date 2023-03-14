// allow lazy loading
function initWikiLink() {
    class WikiLink extends HTMLElement {
        constructor() {
            super();

            var wiki = "";
            if (this.hasAttribute("wiki")) {
                wiki = this.getAttribute("wiki");
            }
            var lang = "";
            if (this.hasAttribute("lang")) {
                lang = this.getAttribute("lang");
            }
            var title = this.getAttribute("title");

            var shadow = this.attachShadow({mode: 'open'});
            var link = document.createElement('a');
            link.href = buildWikiLink(wiki, lang, title);
            link.textContent = this.textContent;
            shadow.appendChild(link);
        }
    }

    customElements.define('wiki-link', WikiLink);
}