# pgtogogen
Pgtogogen is a Postgres entities to Go structures generator with the low-level db connectivity details handled by Jack Christensen's pgx project  (https://github.com/jackc/pgx).
The project was started in late 2014, committed to GitHub in early 2015, and I've kept adding features sporadically even since.

### Status: In Progress
The project is partially functional. As is the case with most GitHub project, use it at your own risk. 

### Installation
To install the tool, run:
 go get github.com/silviucm/pgtogogen
	
### Generating the model package	
Assuming an empty subdirectory named "models" exists in the location where you will run the command:
 pgtogogen -h=localhost -n=mydatabasename -u=mydatabaseuser -pass=mydatabasepassword

### Usage
TODO
	

### License
The MIT License (MIT)

Copyright (c) 2015 Silviu Capota Mera

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
