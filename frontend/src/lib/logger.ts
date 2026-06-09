import { AppService } from "../../bindings/github.com/wanglejiu/llm-proxy";

const logger = {
  debug(msg: string) {
    AppService.LogDebug(msg).catch(() => {});
  },
  info(msg: string) {
    AppService.LogInfo(msg).catch(() => {});
  },
  warn(msg: string) {
    AppService.LogWarn(msg).catch(() => {});
  },
  error(msg: string) {
    AppService.LogError(msg).catch(() => {});
  },
};

export default logger;
