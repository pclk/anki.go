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
# How to send POST request in python with headers and binary file?
requests.get(url, headers={}, data=filedata)

# The five AutoML offerings include {{Vision}}, {{Tables}}, {{Natural Language}}, {{Video Intelligence}}, and {{Translation}}.

# What Google Cloud tools do AI Engineers use?
Deep learning containers 

VM Image

# The four most popular cloud offerings are {{Image}}, {{Natural Language Processing}}, {{Speech}}, and {{Chatbots}}.
These are offered by Google cloud, Azure and AWS.
```
Run `anki ml.md`,

And `anki.go` will create a new file based off it.

> file_name:anki-ml.md
```md
model: Basic

# Note

## Front
How to send POST request in python with headers and binary file?

## Back
requests.get(url, headers={}, data=filedata)

# Note
model: Cloze

## Text
The five AutoML offerings include {{c1::Vision}}, {{c2::Tables}}, {{c3::Natural Language}}, {{c4::Video Intelligence}}, and {{c5::Translation}}.

## Back Extra


# Note

## Front
What Google Cloud tools do AI Engineers use?

## Back
Deep learning containers 

VM Image

# Note
model: Cloze

## Text
The four most popular cloud offerings are {{c1::Image}}, {{c2::Natural Language Processing}}, {{c3::Speech}}, and {{c4::Chatbots}}.

## Back Extra
These are offered by Google cloud, Azure and AWS.
```

`anki.go` will execute the command `apy add-from-file anki-ml.md` to insert the notes into your anki db. It will also return the output and errors of the command.

## Additional features
`-d` allows you to specify decks for the cards you're adding
```sh
anki -d Machine_learning ml.md 
```

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
- Database is NA/locked!
> Close your Anki

## TODO
- Anthropic integration
Step 1: Distill lecture material from pdf, word etc to a nicely formatted markdown file 
> You could upload your pdf to NotebookLM and collect the formatted document, or ask the AI to generate notes based off it.

Step 2: Compare lecture material and your markdown file. Add information about diagrams, images, and other missing information.
> This is the stage where you understand you lecture material.

Step 3: Call anki --ai note.md
This will call anthropic Claude Sonnet 3.5 to generate Cloze deletions from your notes

1. Take the first section and ask Claude to generate.
2. Claude generates output A.
3. No matter what output A is, tell Claude "more detailed please".
4. Claude generates output B.
5. Create file `anki.md`, and append output B to anki.md
6. Provide the next section
7. Claude generates output C.
8. Append output C to anki.md.
9. Repeat step 6 to 8 until completion of note.md

Each section is separated via ## (h2) markdown headers.
