import { mkdir, appendFile } from "fs/promises";
import { join } from "path";

export const PromptLogger = async ({ worktree }) => {
  return {
    "chat.message": async (_input, output) => {
      if (output.message.role !== "user") return;

      const now = new Date();
      const year = now.getFullYear();
      const month = String(now.getMonth() + 1).padStart(2, "0");
      const day = String(now.getDate()).padStart(2, "0");
      const time = now.toLocaleTimeString("th-TH", { hour12: false });

      const logDir = join(worktree, ".opencode", "logs", String(year), month, day);
      const logFile = join(logDir, "prompt.md");

      await mkdir(logDir, { recursive: true });

      const text = output.parts
        .filter((p) => p.type === "text")
        .map((p) => p.text)
        .join("");

      const entry = `## ${time}\n\n${text}\n\n---\n\n`;
      await appendFile(logFile, entry, "utf-8");
    },
  };
};
