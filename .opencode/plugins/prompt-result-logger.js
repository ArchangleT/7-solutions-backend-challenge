import { mkdir, appendFile } from "fs/promises";
import { join } from "path";

const sessions = new Map();

export const PromptResultLogger = async ({ worktree }) => {
  return {
    "chat.message": async (input, output) => {
      const id = input.sessionID ?? "default";
      let session = sessions.get(id);
      if (!session) {
        session = { prompt: "", response: "", startTime: 0, tokens: null, cost: null };
        sessions.set(id, session);
      }

      const text = output.parts
        .filter((p) => p.text != null)
        .map((p) => p.text)
        .join("");

      if (output.message.role === "user") {
        if (text) session.prompt = text;
        session.startTime = session.startTime || Date.now();
      } else if (output.message.role === "assistant") {
        if (text) session.response = (session.response || "") + text;
        const msg = output.message;
        if (msg.cost != null) session.cost = msg.cost;
        if (msg.tokens) session.tokens = msg.tokens;
      }
    },

    event: async ({ event }) => {
      if (event.type !== "session.idle") return;

      const now = new Date();
      const year = now.getFullYear();
      const month = String(now.getMonth() + 1).padStart(2, "0");
      const day = String(now.getDate()).padStart(2, "0");
      const time = now.toLocaleTimeString("th-TH", { hour12: false });

      const id = event.properties.sessionID ?? "default";
      const session = sessions.get(id);

      const elapsed = session?.startTime
        ? ((Date.now() - session.startTime) / 1000).toFixed(1)
        : "N/A";

      let tokenCredit = "N/A";
      let difficulty = "N/A";

      if (session?.tokens) {
        const t = session.tokens;
        const parts = [
          `Input: ${t.input.toLocaleString()}`,
          `Output: ${t.output.toLocaleString()}`,
        ];
        if (t.reasoning) parts.push(`Reasoning: ${t.reasoning.toLocaleString()}`);
        if (t.cache?.read || t.cache?.write) {
          parts.push(`Cache: ${t.cache.read.toLocaleString()}/${t.cache.write.toLocaleString()}`);
        }
        tokenCredit = parts.join(", ");
        if (session.cost != null) {
          tokenCredit += ` | $${session.cost.toFixed(4)}`;
        }

        const total = t.input + t.output + (t.reasoning || 0);
        if (total > 10000) difficulty = "ยาก";
        else if (total > 3000) difficulty = "ปานกลาง";
        else difficulty = "ง่าย";
      }

      const promptSummary = session?.prompt
        ? session.prompt.length > 120
          ? session.prompt.slice(0, 120) + "..."
          : session.prompt
        : "N/A";

      const responseSummary = session?.response
        ? session.response.length > 500
          ? session.response.slice(0, 500) + "..."
          : session.response
        : "-";

      const logDir = join(worktree, ".opencode", "logs", String(year), month, day);
      const logFile = join(logDir, "prompt_result.md");

      await mkdir(logDir, { recursive: true });

      const entry = [
        `## ${time}`,
        "",
        `**Prompt:** ${promptSummary}`,
        `**เวลาที่ใช้:** ${elapsed} วินาที`,
        `**Tech Stack:** Go, MongoDB, JWT, net/http`,
        `**Token/Credit:** ${tokenCredit}`,
        `**ระดับความยาก:** ${difficulty}`,
        "",
        "### Summary",
        "",
        responseSummary,
        "",
        "---",
        "",
      ].join("\n");

      await appendFile(logFile, entry, "utf-8");
      sessions.delete(id);
    },
  };
};
