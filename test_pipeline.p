detail(topic):
	We are writing a book about [topic]. List 3 chapters with titles only, numbered.

flesh-out-chapter:
	Expand this chapter into a detailed paragraph.

book(topic):
	topic -> outlined (detail) -> chapters (map(chapters, flesh-out-chapter))

@book(blockchain)
