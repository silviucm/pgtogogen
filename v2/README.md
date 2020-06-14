# pgtogogen
Pgtogogen is a Postgres-entities-to-Go structures generator with the low-level db connectivity details handled by Jack Christensen's pgx project  (https://github.com/jackc/pgx).
The project was started in late 2014, committed to GitHub in early 2015, and I've kept adding features sporadically ever since.

### Status: In Progress
The project is partially functional. As is the case with most GitHub projects, use it at your own risk. 

### Compatibility

pgtogogen is compatible with github.com/jackc/pgx/v3. For github.com/jackc/pgx/v2 compatibility, you will need to use the pgxV2 branch of pgtogogen

pgtogogen/v2 version supports a minimum of github.com/jackc/pgx/v4. It will not work with earlier versions.

### Installation

To install the github.com/jackc/pgx/v3 compatible tool, run:
```bash
 go get -u github.com/silviucm/utils
 go get -u github.com/silviucm/pgtogogen
```

To install the github.com/jackc/pgx/v4 compatible tool, run:
```bash
 go get -u github.com/silviucm/utils
 go get -u github.com/silviucm/pgtogogen/v2
```
	
### Generating the model package	
Assuming an empty subdirectory named "models" exists in the location where you will run the command:
```bash
 pgtogogen -h=localhost -n=mydatabasename -u=mydatabaseuser -pass=mydatabasepassword
```

### Usage

Initialize the database (do it in the main init() function or as soon as possible in the main() function):

```go
 	// the models package is the one generated by the tool (e.g. github.com/yourproject/models)
	var poolMaxConnections int = 100
	_, err := models.InitDatabaseMinimal("192.168.0.1", 5432, "hellouser", "hellopass", "mydatabase", poolMaxConnections)

	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Println("OK.")
	}
```
For a comprehensive usage article, see: 

[https://www.cmscomputing.com/articles/programming/generate-go-entities-from-postgres-tables-views](https://www.cmscomputing.com/articles/programming/generate-go-entities-from-postgres-tables-views)

### License
The MIT License (MIT)

Copyright (c) 2015,2016,2017 Silviu Capota Mera

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.