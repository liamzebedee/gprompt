# nyt.p
# goal: run a tiny “NYT” as a living system:
# - gather info
# - draft articles in parallel
# - edit
# - publish
# - repeat forever

nyt:
	loop(30m):
		run(daily_cycle)

daily_cycle:
	"cycle" -> briefs(fetch_briefs) -> pitches(make_pitches) -> drafts(map(pitches, write_draft)) -> edited(edit_pack) -> publish_all(publish_pack)

fetch_briefs:
	sources:
		https://www.reuters.com
		https://apnews.com
		https://www.bbc.com/news
		https://www.ft.com
		https://www.economist.com
		https://news.ycombinator.com
	briefs = map(sources, fetch_and_extract)
	merge(briefs)

fetch_and_extract(url):
	Read [url]. Extract:
	- 10 bullet “facts/events” with dates
	- 10 bullet “what changed since yesterday”
	- 10 bullet “things to watch”
	Return as compact plaintext.

make_pitches:
	You are editor-in-chief.
	From these briefs, propose 5 article pitches.
	Each pitch must be:
	- title
	- one-sentence angle
	- target reader
	- 5 key facts to include
	- which source lines support each fact
	Return a list named `pitches`.

write_draft(pitch):
	role:
		writer
	style:
		clear, short paragraphs, no fluff
	Write one article from [pitch].
	Constraints:
	- 700–1200 words
	- include a “what we know / what we don’t” section
	- include dates explicitly
	Save to drafts/[pitch.title].md
	Return the path.

edit_pack:
	role:
		editor
	Input is a list of draft paths.
	For each draft:
	- fix structure
	- remove weak claims
	- enforce that every key factual claim cites the brief line it came from
	- add a 2-sentence lede
	Save to edited/[same_name].md
	Return list of edited paths.

publish_pack:
	role:
		publisher
	Input is list of edited paths.
	For each:
	- generate slug
	- generate 160-char description
	- write to site/posts/[date]-[slug].md
	- append to site/index.md
	Return ok.

loop(period):
	Run forever.
	Every [period], execute the body.

run(fn):
	Execute function [fn].

#merge(xs):
#	Join list items into one block.

