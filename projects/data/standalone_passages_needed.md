# Standalone Passages — Text Verification Needed

The entries in `standalone_passages.json` were reconstructed from modern KJV text.
Replace each `"content"` field with the exact wording Schmidt quotes in the Lexicon.

---

## Biblical passages (edition: kjv_1611)

| Citation ID | Key | Reference | Current (reconstructed) text |
|-------------|-----|-----------|------------------------------|
| 153785 | Punishment | Genesis III, 16 | "Unto the woman he said, I will greatly multiply thy sorrow and thy conception; in sorrow thou shalt bring forth children; and thy desire shall be to thy husband, and he shall rule over thee." |
| 184626 | Spot1 | Jeremiah XIII, 23 | "Can the Ethiopian change his skin, or the leopard his spots? then may ye also do good, that are accustomed to do evil." |
| 144087 | Philip | Acts XXI, 9 | "And the same man had four daughters, virgins, which did prophesy." |
| 185243 | Spurn2 | Acts IX, 5 | "And he said, Who art thou, Lord? And the Lord said, I am Jesus whom thou persecutest: it is hard for thee to kick against the pricks." |
| 67473 | First-born | Exodus XI, 5 | "And all the firstborn in the land of Egypt shall die, from the firstborn of Pharaoh that sitteth upon his throne, even unto the firstborn of the maidservant that is behind the mill; and all the firstborn of beasts." |
| 176206 | Shuttle | Job VII, 6 | "My days are swifter than a weaver's shuttle, and are spent without hope." |
| 227029 | Woe | Ecclesiastes X, 16 | "Woe to thee, O land, when thy king is a child, and thy princes eat in the morning!" |
| 126931 | Nabuchadnezzar | Daniel IV, 33 | "The same hour was the thing fulfilled upon Nebuchadnezzar: and he was driven from men, and did eat grass as oxen, and his body was wet with the dew of heaven, till his hairs were grown like eagles' feathers, and his nails like birds' claws." |
| 12185 | Babe | Matthew XI, 25 | "At that time Jesus answered and said, I thank thee, O Father, Lord of heaven and earth, because thou hast hid these things from the wise and prudent, and hast revealed them unto babes." |
| 110926 | Locusts | Matthew III, 4 | "And the same John had his raiment of camel's hair, and a leathern girdle about his loins; and his meat was locusts and wild honey." |
| 168112 | Save1 | 1 Corinthians VII, 14 | "For the unbelieving husband is sanctified by the wife, and the unbelieving wife is sanctified by the husband: else were your children unclean; but now are they holy." |
| 220701 | Weaver | Samuel XVII, 7 | "And the staff of his spear was like a weaver's beam; and his spear's head weighed six hundred shekels of iron: and one bearing a shield went before him." |
| 130906 | North | Isaiah XIV, 13 | "For thou hast said in thine heart, I will ascend into heaven, I will exalt my throne above the stars of God: I will sit also upon the mount of the congregation, in the sides of the north:" |
| 2811 | Admonish | Hebrews VIII, 5 | "Who serve unto the example and shadow of heavenly things, as Moses was admonished of God when he was about to make the tabernacle: for, See, saith he, that thou make all things according to the pattern shewed to thee in the mount." |
| 166523 | Sack-cloth | Esther IV, 1 | "When Mordecai perceived all that was done, Mordecai rent his clothes, and put on sackcloth with ashes, and went out into the midst of the city, and cried with a loud and a bitter cry;" |
| 170222 | Security | Proverbs XI, 15 | "He that is surety for a stranger shall smart for it: and he that hateth suretiship is sure." |
| 225386 | Wisdom | Proverbs I, 20 | "Wisdom crieth without; she uttereth her voice in the streets:" |
| 26 | A-hungry | Mark II, 25 | "And he said unto them, Have ye never read what David did, when he had need, and was an hungred, he, and they that were with him?" |
| 105627 | Lazarus | Luke XVI, 20 | "And there was a certain beggar named Lazarus, which was laid at his gate, full of sores," |
| 232325 | Young | Luke XV, 12 | "And the younger of them said to his father, Father, give me the portion of goods that falleth to me. And he divided unto them his living." |

---

## Also still needed (no text entered yet)

| Citation ID | Key | Reference | Notes |
|-------------|-----|-----------|-------|
| 2808 | Adonis | Pliny XIX, 19, 1 | Adonis-gardens passage; use Holland (1601) or Bostock & Riley (1855) |
| 10969 | Autolycus | Homer Od. XIX, 394 | Autolycus naming; use Chapman's Homer (1616) |
| 76689 | Gimmal | Edward III I, 2, 29 | "gimmal" ring line; any public domain edition |

---

## How to update a passage

Edit the `"content"` field for the relevant entry in
`projects/data/standalone_passages.json`, then run:

```
go run ./cmd/build -step standalone
go run ./cmd/build -step citations
```
