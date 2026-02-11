book(topic):
	topic -> brief (book-idea) -> chapter-outline (generate-chapter-index) -> chapters (map(chapters, flesh-out-chapter)) -> final (concat)

book-idea(topic):
	We are writing a book about [topic]. Generate a briefer on what it should cover and why it's good.

generate-chapter-index:
	From this briefer on a book, generate an index of chapters - 1 per line with a title. Max 5 chapters.
	
flesh-out-chapter:
	Expand this chapter into a title, 2 paragraphs, and conclusion. 
	Save it to chapters/IDX.md

concat:
	Take all the chapters of this book in `chapters/*.md` and put it into one markdown file `book.md`, adding structure as needed.

@book(blockchain)