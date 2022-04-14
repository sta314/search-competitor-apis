# Non Official APIs for Google

This project mainly focuses on getting information from Google's services which don't provide any official API. 
#### Programming Language

<img src="/docs/svg/go-logo.svg" alt="drawing" width="200"/>

Project is completely written with Go due to its efficient architecture, especially for concurrency.
#### DBMS
<img src="/docs/svg/ch-logo.svg" alt="drawing" width="200"/>

ClickHouse is used for its column based, very fast structure which is suitable for our case.

## List of provided APIs
Note: Detailed explanation of each API is available in each subprojects folder 
#### Keyword Suggestion API

[This subproject](google-suggestions) focuses on providing suggestions with desired depth for a given keyword.
```
input : example
output :
...
  {
    "suggested_keyword": "example product review",
    "base_keywords": [
      "example",
      "example review"
    ],
    "level": 2
  },
...
```
#### SERP API

[This subproject](google-serp) focuses on getting Google search results in useful form.
```
input : product
output:
...
  {
    "url": "https://www.example.com/",
    "rank": 20,
    "title": "page title here",
    "description": "page description here"
  }
  ...
```

#### Competitor Analysis API

[This subproject](google-competitor) uses the data has been gathered from SERP API and returns possible competitors for a given keyword.
```
input : example
output:
...
  {
    "url_domain": "competitor-b.example",
    "score": 26812,
    "place": 3
  }
  ...
```

## Common information for all APIs

API server starts on: 
```sh
localhost:8080
```
ClickHouse server assumed to be running at API startup and assumed to have following properties:
```sh
address = 127.0.0.1:9000
database = "default"
username = "default"
password = ""
```
Every subproject has its own **Swagger** documentation, check it to learn more about them and use a simple UI:
```sh
localhost:8080/swagger/index.html
```
