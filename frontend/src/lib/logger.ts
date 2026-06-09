import { AppAPI } from "../services";

/** 将结构化数据序列化为字符串，发送到后端日志系统，同时输出到浏览器控制台 */
function format(tag: string, msg: string, data?: unknown): string {
  if (data === undefined) return `[${tag}] ${msg}`;
  try {
    return `[${tag}] ${msg} ${JSON.stringify(data)}`;
  } catch {
    return `[${tag}] ${msg} [序列化失败]`;
  }
}

const logger = {
  debug(msg: string, data?: unknown) {
    const text = format("DEBUG", msg, data);
    console.debug(text);
    AppAPI.logDebug(text);
  },
  info(msg: string, data?: unknown) {
    const text = format("INFO", msg, data);
    console.info(text);
    AppAPI.logInfo(text);
  },
  warn(msg: string, data?: unknown) {
    const text = format("WARN", msg, data);
    console.warn(text);
    AppAPI.logWarn(text);
  },
  error(msg: string, data?: unknown) {
    const text = format("ERROR", msg, data);
    console.error(text);
    AppAPI.logError(text);
  },
};

export default logger;
