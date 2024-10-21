# anki.go

> Convert .md files to anki cards!
<!--toc:start-->
- [Usage](#usage)
- [Additional features](#additional-features)
- [Installation](#installation)
- [Frequent problems](#frequent-problems)
<!--toc:end-->

## Usage

Format your .md files as such:

> file_name:ml.md
```md
# What is the equation for linear regression?
y = a + bx + e 

where a is intercept

b is slope of line 

e is error term

# What modelling technique should you use if your variable is continuous?
Linear regression

# What is the most common method to obtain the best fit line of a linear regression?
Least Square method. It minimizes the sum of the squares of the vertical deviation from each data point to the line.

# Why is the values of the Least Square method absolute?
The vertical deviation from each data point to the line is first squared, which turns negative into positive values.
```
Run `anki ml.md`,

And `anki.go` will create a new file based off it.

> file_name:anki-ml.md
```md
model: Basic

# Note

## Front
What is the equation for linear regression?

## Back
y = a + bx + e 

where a is intercept

b is slope of line 

e is error term

# Note

## Front
What modelling technique should you use if your variable is continuous?

## Back
Linear regression

# Note

## Front
What is the most common method to obtain the best fit line of a linear regression?

## Back
Least Square method. It minimizes the sum of the squares of the vertical deviation from each data point to the line.

# Note

## Front
Why is the values of the Least Square method absolute?

## Back
The vertical deviation from each data point to the line is first squared, which turns negative into positive values.
```

`anki.go` will execute the command `apy add-from-file anki-ml.md` to insert the notes into your anki db. It will also return the output of the command.

## Additional features
`-d` allows you to specify default decks for the files you're importing.
```sh
anki -d Machine_learning ml.md 
```
This will execute the command `apy add-from-file anki-xxx.md -d Machine_learning`

`anki c` will execute a cleanup process to delete all anki-xxx.md files in the working directory.

## Installation
Clone this repo:
```sh
git clone https://github.com/pclk/anki.go
```
Install with go:
```sh 
go install anki.go
```
Install apy:
```sh
pipx install apy
```
Use:
```sh 
anki test/test.md
```

## Frequent problems
- Database is locked
> Close your anki.
