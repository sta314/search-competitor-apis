# Competitor Analysis API

This subproject uses the data has been gathered from SERP API and returns possible competitors for a given keyword.


## What is the logic?

Scores each domain using the ranking in results for all the keywords which given domain is in results.

### How to call?
```sh
GET localhost:8080/get_competitors?keyword=example&limit=10
```

### What does response look like?
Response is *json* containing all results as an array.
```
...
  {
    "url_domain": "competitor-a.example",
    "score": 29179,
    "place": 2
  },
  {
    "url_domain": "competitor-b.example",
    "score": 26812,
    "place": 3
  },
  {
    "url_domain": "competitor-c.example",
    "score": 26119,
    "place": 4
  },
  ...
```
**url_domain** : Domain of ranked URL\
**score** : Total score for this domain\
**place** : Rank among all domains

### Parameters
+ **keyword** : Domain of the URL that competitors will searched
+ **limit** : Amount of desired results

### Example data

Previously executed query using SERP DB filled with [this keyword list](/docs/data/keyword_list.csv) and **keyword = example** can be reached from [here](/docs/data/competitor)
