import { Page } from "@playwright/test";
import { XMLParser } from "fast-xml-parser";
import { logger } from "./logger";

const urlPrefix = "https://www.perseus.tufts.edu/hopper";
export const firstLetterOfLexicon = "text?doc=Perseus%3Atext%3A1999.03.0079";
export const lastLetterOfLexicon = null;

export const processEntries = async (
  page: Page,
  test: any,
  options: {
    chunkSize?: number;
    timeoutBetweenChunks?: number;
    onChunkComplete?: (chunk: EntryItem[]) => Promise<void>;
  } = {}
): Promise<EntryItem[]> => {
  const {
    chunkSize = 10,
    timeoutBetweenChunks = 10000,
    onChunkComplete = async () => {},
  } = options;

  let previousWord: string | null = null;
  let currentWord: string = firstLetterOfLexicon;
  let nextWord: string | null = null;
  test.setTimeout(620000);
  await page.goto(`${urlPrefix}/${currentWord}`);
  let currentCount = 0;
  const returnValues: EntryItem[] = [];
  let currentChunk: EntryItem[] = [];

  //only get limited number of values in development
  while (currentWord !== lastLetterOfLexicon) {
    const xmlPath = await page
      .locator("#footer_nav")
      .getByRole("link", { name: "view as XML" })
      .getAttribute("href");

    // if (previousLetter !== null) {
    //   previousLetter = await page.locator('#footer_nav').getByRole("link", { name: 'previous' }).getAttribute('href')
    // }

    nextWord = await page
      .locator("#footer_nav")
      .getByRole("link", { name: "next" })
      .getAttribute("href");

    const xmlFile = await page.goto(`${urlPrefix}/${xmlPath}`);
    const xmlBody = await xmlFile?.body();
    currentCount++;

    if (xmlBody) {
      const parsedXml: any = new XMLParser().parse(xmlBody);

      const returnValue: EntryItem = {
        entry: { ...parsedXml },
        previousWord: previousWord || null,
        currentWord: currentWord || "",
        nextWord: nextWord || null,
      };

      returnValues.push(returnValue);
      currentChunk.push(returnValue);

      // Process chunk if we've reached the chunk size
      if (currentChunk.length >= chunkSize) {
        logger.info(`Processed chunk of ${currentChunk.length} entries`);
        await onChunkComplete(currentChunk);
        currentChunk = [];

        // Wait between chunks
        logger.info(
          `Waiting for ${
            timeoutBetweenChunks / 1000
          } seconds before next chunk...`
        );
        await new Promise((resolve) =>
          setTimeout(resolve, timeoutBetweenChunks)
        );
      }
    } else {
      logger.warn("no xml body detected");
    }

    await page.goBack();

    await page
      .locator("#footer_nav")
      .getByRole("link", { name: "next" })
      .click();
    previousWord = currentWord;
    currentWord = nextWord || "";
  }

  // Process any remaining entries in the last chunk
  if (currentChunk.length > 0) {
    logger.info(`Processing final chunk of ${currentChunk.length} entries`);
    await onChunkComplete(currentChunk);
  }

  return returnValues;
};
