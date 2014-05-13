package webl

import (
  "fmt"
  "sync"
  "net/http"
  "code.google.com/p/go.net/html"
  "io"
  "gopkg.in/fatih/set.v0"
  "io/ioutil"
)

func Crawl(domainName string) {
  INFO.Println(fmt.Sprintf("About to crawl: %s", domainName))
  var wg sync.WaitGroup
  wg.Add(1)
  alreadyProcessed := set.New()
  url := toUrl(domainName,"")
  name := toFriendlyName(url)

  addDomain(&Resource{ Name: name, Url: url })
  go fetchResource(name,url,alreadyProcessed,&wg)
  TRACE.Println("Wait...")
  wg.Wait();
  TRACE.Println("Done waiting.")

  savedResource := LoadDomain(domainName,true)
  writeSitemap(&savedResource, fmt.Sprintf("./tmp/%s.sitemap.xml", domainName))
  INFO.Println(fmt.Sprintf("Done crawing: %s", domainName))
}

func fetchResource(domainName string, currentUrl string, alreadyProcessed *set.Set, wg *sync.WaitGroup) {
  defer wg.Done()
  if alreadyProcessed.Has(currentUrl) {
    TRACE.Println(fmt.Sprintf("Duplicate (skipping): %s", currentUrl))
  } else if shouldProcessUrl(domainName,currentUrl) {
    saveResource(&Resource{ Name: toFriendlyName(currentUrl), Url: currentUrl })    
    alreadyProcessed.Add(currentUrl)
    TRACE.Println(fmt.Sprintf("Fetch: %s", currentUrl))
    resp, err := http.Get(currentUrl)
    should_close_resp := true

    if err != nil {
      WARN.Println(fmt.Sprintf("UNABLE TO FETCH %s, due to %s", currentUrl, err))
    } else {
      contentType := resp.Header.Get("Content-Type")
      lastModified := resp.Header.Get("Last-Modified")
      TRACE.Println(fmt.Sprintf("Done Fetch (%s %s): %s",contentType, resp.Status, currentUrl))
      saveResource(&Resource{ Name: toFriendlyName(currentUrl), Url: currentUrl, Type: contentType, Status: resp.Status, StatusCode: resp.StatusCode, LastModified: lastModified })
      if IsWebpage(contentType) {
        if (!shouldProcessUrl(domainName,resp.Request.URL.String())) {
          TRACE.Println(fmt.Sprintf("Not following %s, as we redirected to a URL we should not process %s", currentUrl,resp.Request.URL.String()))
        } else {
          should_close_resp = false
          wg.Add(1);
          go analyzeResource(domainName, currentUrl, resp, alreadyProcessed, wg)
        }
      }
    }
    if should_close_resp {
      defer resp.Body.Close()
      defer io.Copy(ioutil.Discard, resp.Body)
    }
  } else {
    TRACE.Println(fmt.Sprintf("Skipping: %s", currentUrl))
  }
}

func analyzeResource(domainName string, currentUrl string, resp *http.Response, alreadyProcessed *set.Set, wg *sync.WaitGroup) {
  defer wg.Done()
  defer resp.Body.Close()
  defer io.Copy(ioutil.Discard, resp.Body)

  TRACE.Println(fmt.Sprintf("Analyze: %s", currentUrl))
  tokenizer := html.NewTokenizer(resp.Body)
  for { 
    token_type := tokenizer.Next() 
    if token_type == html.ErrorToken {
      if tokenizer.Err() != io.EOF {
        WARN.Println(fmt.Sprintf("HTML error found in %s due to ", currentUrl, tokenizer.Err()))  
      }
      return     
    }       
    token := tokenizer.Token()
    switch token_type {
    case html.StartTagToken, html.SelfClosingTagToken: // <tag>
      path := resource_path(token)
      if path != "" {
        wg.Add(1)
        nextUrl := toUrl(domainName,path)
        saveEdge(domainName,currentUrl,nextUrl)
        go fetchResource(domainName,nextUrl,alreadyProcessed,wg)
      }
    }
  }
  TRACE.Println(fmt.Sprintf("Done Analyze: %s", currentUrl))  
}
