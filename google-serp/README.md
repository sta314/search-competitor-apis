# SERP API

This subproject focuses on getting Google search results in useful form.

### How to call?
```sh
GET localhost:8080/get_serp?keyword=example&hl=en&gl=us&limit=10
```
### What does response look like?
Response is *json* containing all results as an array.
```
  {
    "url": "https://www.example.com/",
    "rank": 1,
    "title": "Example",
    "description": "Example page description"
  },
...
```
**url** : URL of referred page\
**rank** : Organic rank of the page in result page\
**title** Title of referred page\
**description** Description of referred page

### Parameters
+ **keyword** : Keyword to search
+ **hl** : Host language code to restrict results
+ **gl** : Two-letter country code to restrict results
+ **limit** : Amount of desired results
+ **load** : Check database for cached results if set to true
+ **save** : Save results to the database if set to true
+ **proxy** : Proxy to use for request

### Table for the database

google_serp_results

| keyword  | hl | gl | timestamp | url | rank | title | description |
| --- | --- | --- | --- | --- | --- | --- | --- |

google_serp_results_meta

| keyword | hl | gl | timestamp | limit |
| --- | --- | --- | --- | --- |

### Example data

Previously collected data using [this keyword list](/docs/data/keyword_list.csv) can be reached from [here](/docs/data/serp)
