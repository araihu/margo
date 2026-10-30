package site

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/chromedp"
)

func TestEditorialBodyHistoryMetadataRejectsStaleResponse(t *testing.T) {
	executable := installedSiteTestChromium()
	if executable == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	result := editorialFixture(t, AssetsLocal)
	root := t.TempDir()
	for _, artifact := range result.Artifacts {
		name := filepath.Join(root, filepath.FromSlash(artifact.Path))
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, artifact.Content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(http.FileServer(http.Dir(root)))
	defer server.Close()
	allocator, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(executable)...)
	defer cancelAllocator()
	browser, cancelBrowser := chromedp.NewContext(allocator)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browser, 30*time.Second)
	defer cancel()
	home, _ := json.Marshal(string(configArtifact(t, result, "index.html")))
	post, _ := json.Marshal(string(configArtifact(t, result, "posts/index.html")))
	if err := chromedp.Run(ctx, chromedp.Navigate(server.URL+"/posts/"), chromedp.Poll(`window.htmx && window.Alpine`, nil), chromedp.Evaluate(`(() => {
 const home=`+string(home)+`;
 const stale=`+string(post)+`;
 const restore={target:document.body,text:home,title:document.title};
 document.dispatchEvent(new CustomEvent('htmx:before:swap',{detail:{ctx:restore,tasks:[]}}));
 const main=new DOMParser().parseFromString(home,'text/html').querySelector('#main-content');
 document.querySelector('#main-content').replaceWith(main);
 history.replaceState(history.state,'','/');
 document.dispatchEvent(new CustomEvent('htmx:after:swap',{detail:{ctx:restore}}));
 if(restore.title) document.title=restore.title;
 if(!document.title.startsWith('Home') || !document.querySelector('link[rel="canonical"]')?.href.endsWith('/'))throw Error('history metadata not restored: '+document.title);
 const late={target:document.body,text:stale,title:'Posts · Margo'};
 document.dispatchEvent(new CustomEvent('htmx:before:swap',{detail:{ctx:late,tasks:[]}}));
 document.dispatchEvent(new CustomEvent('htmx:after:swap',{detail:{ctx:late}}));
 if(late.title) document.title=late.title;
 if(!document.title.startsWith('Home') || !document.querySelector('link[rel="canonical"]')?.href.endsWith('/'))throw Error('stale metadata overwrote restored route');
 return true;
 })()`, nil)); err != nil {
		t.Fatal(err)
	}
}
