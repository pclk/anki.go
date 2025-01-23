Guide to markdown to note conversion
Start: anki.go

main function calls run(), processing -c and -d flags which are usually not used.
What's important in markdown to note conversion is the cmd.CreateNote()

CreateNote() takes input markdown file, and calls converter.ConvertToAnki()

```
converter/
  ├── converter.go      (main package file with core conversion logic)
  ├── logger.go         (logging functionality)
  ├── template.go       (template processing)
  ├── section.go        (section handling)
  ├── note/
  │   ├── basic.go      (basic note type)
  │   └── cloze.go      (cloze note type)
  └── processor/
      └── line.go       (line processing logic)
```


ConvertToAnki() opens input file and extracts deck name.

Extract deckname goes through each line until it meets a line with "# deck:" as a prefix.
