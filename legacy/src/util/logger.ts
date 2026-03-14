import pino from "pino";


export const logger = pino({
  name: "dictionary-scraper",
  transport: {
    target: "pino-pretty",
    options: {
      singleLine: true,
    },
  },
});
