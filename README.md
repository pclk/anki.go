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
# deck: ml

How to send POST request in python with headers and binary file?
requests.get(url, headers={}, data=filedata)

The five AutoML offerings include -Vision-, -Tables-, -Natural Language-, -Video Intelligence-, and -Translation-.

What Google Cloud tools do AI Engineers use?
Deep learning containers
VM Image
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

### Basic cards
```md
hello?
are you there?
```
-->
```md
model: Basic

# Note

## Front
hello?

## Back
are you there?
```
> A line ending with a question mark ? signifies the front of a Basic card.
> Subsequent lines, until a blank line is encountered, form the back of the Basic card.

### Cloze deletion cards
#### Single-lined consecutive clozes
```md
this is a -single- line -cloze- deletion -card-.
```
-->
```md
# Note
model: Cloze

## Text
this is a {{c1::single}} line {{c2::cloze}} deletion {{c3::card}}.

## Back Extra

```

#### Single-lined consecutive clozes with back extra
```md
this is a -single- line -cloze- deletion -card-.
> I love to use AI to generate the back extra to give extra details about the card.
```
-->
```md
# Note
model: Cloze

## Text
this is a {{c1::single}} line {{c2::cloze}} deletion {{c3::card}}.

## Back Extra
I love to use AI to generate the back extra to give extra details about the card.

```

#### Single-lined grouped clozes
```md
this is a -single- line 1.-cloze- deletion 1.-card-.
```
-->
```md
# Note
model: Cloze

## Text
this is a {{c1::single}} line {{c2::cloze}} deletion {{c2::card}}.

## Back Extra

```
> Syntax: n.-text-, n.-text-, where `n` is the same number. `n` does not represent the card number ({{c`n`::text}}), but rather a group identifier that groups the clozes.

#### Multi-lined clozes
```md
this is a -multi- line 1.-cloze- deletion 1.-card-.
The text will -no\-longer- be part of this line if there's a
2.-new- blank 2.-line- after.
```
-->
```md
# Note
model: Cloze

## Text
this is a {{c1::multi}} line {{c2::cloze}} deletion {{c2::card}}.
The text will {{c3::no-longer}} be part of this line if there's a
{{c4::new}} blank {{c4::line}} after.

## Back Extra

```

### Sections
```md
## Water

The water has the possibility of getting things -wet-.

Why are things considered wet?
Water molecules fill up the holes of the item.
```
-->
```md
# Note
model: Cloze

## Text
Section: Water

The water has the possibility of getting things {{c1::wet}}.

## Back Extra


# Note

## Front
Seciton: Water

Why are things considered wet?

## Back
Water molecules fill up the holes of the item.
```
> Automatically prepends the current section prefix, followed by `\n\n`, to the front of the Basic/Cloze card during the conversion.
> Sections are denoted by h2 and above headers.

#### Overriding Sections
```md
## Water
This section is about -water-.

## Fire
However, this section has been overriden with -Fire-.

very -spicy-.

## Clear section
Subsequent cards no longer have any -section prefix-.
```
-->
```md
# Note
model: Cloze

## Text
Section: Water

This section is about {{c1::water}}.

## Back Extra


# Note
model: Cloze

## Text
Section: Fire
However, this section has been overriden with {{c1::Fire}}.

## Back Extra


# Note
model: Cloze

## Text
Section: Fire

very {{c1::spicy}}.

## Back Extra


# Note
model: Cloze

## Text
Subsequent cards no longer have any {{c1::section prefix}}.

## Back Extra

```

#### Nested sections
```md
## Water 
this section is about -water-.

### Liquid
in the liquid form, water -can- (can or not) be compressed.

#### Carbonation
carbonated water is still -liquid water-, just with dissolved carbon dioxide.

### Gas
in the gaseous form, steam can be -compressed-.


## Fire
this section is about -fire-.
```
-->
```md
# Note
model: Cloze

## Text
Section: Water

this section is about {{c1::water}}.

## Back Extra


# Note
model: Cloze

## Text
Section: Water
Sub-section: Liquid

in the liquid form, water {{c1::can}} (can or not) be compressed.

## Back Extra


# Note
model: Cloze

## Text
Section: Water
Sub-section: Liquid
Sub-section: Carbonation

carbonated water is still {{c1::liquid water}}, just with dissolved carbon dioxide.

## Back Extra


# Note
model: Cloze

## Text
Section: Water
Sub-section: Gas

in the gaseous form, steam can be {{c1::compressed}}.

## Back Extra



# Note
model: Cloze

## Text
Section: Fire

this section is about {{c1::fire}}.

## Back Extra

### Comments
```md
# test
test test any sort of
comments
should be able to
- be used all before the deck lols

offerings
# deck: test

## Cloud Services
The five AutoML offerings include -Vision-, -Tables-, -Natural Language-, -Video Intelligence-, and -Translation-.
```
-->
```md

# Note
model: Cloze

## Text
The four most popular cloud offerings are {{c1::Image}}, {{c2::Natural Language Processing}}, {{c3::Speech}}, and {{c4::Chatbots}}.

## Back Extra
These are offered by Google cloud, Azure and AWS.
```
> All text before `# deck:` or `# Deck:` will be ignored.
## Additional features
`-d` allows you to specify decks for the cards you're adding, if you haven't defined the deck at the start.
```sh
anki -d Machine_learning ml.md 
```
vs
```md
# deck: Machine_learning

basic flashcard? yes
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
> Close your Anki and rerun the command. Because we're using `apy` to directly modify the database, we need to close anki to prevent corruption.

## TODO
- use command `anki` to open an editor with the defined default template. This will make it easy to have a system prompt at the start of each deck, which can allow AI to perform actions on the code easily.
- create notes. then, ai give feedback on each section, which is defined in each heading
- switch 1.-text- to -text-.1
- card templates.

```md
# deck: Study

definition of: What is {}?
{}
example: {} examples of {} are -{,}-.


definition of Algorithm
A step-by-step procedure for solving a problem

definition of Binary Search
An efficient search algorithm that works on sorted arrays

example 3 | Binary Tree
BST, AVL Tree, Red-Black Tree
```
-->
```md
What is Algorithm?
A step-by-step procedure for solving a problem

What is Binary Search?
An efficient search algorithm that works on sorted arrays

3 examples of Binary Tree are -BST-, -AVL Tree-, Red\-Black Tree
```
> Syntax: a template definition is a line which is before any cards, starts with words (does not contain {}) followed by colon, followed by the template content (must contain at least 1 {}).
> Usage of templates by writing the template name, followed by a space then the replacement text. Delimited by | or new line.
> Inputting a character within a placeholder definition surrounded with cloze markers (-{,}-) will delimit the replacement text by the character.

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

TODO2: design a markdown file structure, such that configurations like the .env, model, conversation file can be stated at the top. then, for every h1 or whatever, loop through and answer those questions in the h1. 
```markdown
.env: ./.env
conv: my_conversation.json

# system prompt 
bleh

# document
./test/'Week 6 Reinforcement Learning.pdf'

# prompt
Summarize this pdf. make sure its good ok?

# prompt
Not good enough. break it up into 3 sections which we will go through one by one.

# prompt
Let's start with the 1st section. make sure its good.

# prompt
Okay, 2nd section.

# prompt
Okay, 3rd section.
```

after anki -ai this_prompting_file.md:
```markdown
.env: ./.env
conv: my_conversation.json

# system prompt 
bleh

# document
./test/'Week 6 Reinforcement Learning.pdf'

# prompt
Summarize this pdf. make sure its good ok?

I'll help guide you through a detailed study session of these reinforcement learning notes. Let's break it down into key sections:

1. Introduction to Reinforcement Learning:
- It's a type of machine learning where an agent learns to make decisions by interacting with an environment
- The agent takes actions to maximize rewards in specific situations
...

# prompt
Not good enough. break it up into 3 sections which we will go through one by one.

ok. here we go.
section 1 from ... to ...
section 2 from ... to ...
section 3 from ... to ...
do you need any other help?

# prompt
Let's start with the 1st section. make sure its good.

ok. ...(insert 1st section study notes)

# prompt
Okay, 2nd section.


ok. ...(insert 2nd section study notes)

# prompt
Okay, 3rd section.


ok. ...(insert 3rd section study notes)
```
