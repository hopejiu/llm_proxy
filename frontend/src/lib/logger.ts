import { AppAPI } from "../services";

const logger = {
  debug(msg: string) { AppAPI.logDebug(msg); },
  info(msg: string) { AppAPI.logInfo(msg); },
  warn(msg: string) { AppAPI.logWarn(msg); },
  error(msg: string) { AppAPI.logError(msg); },
};

export default logger;
