package notifierimpl

import (
	domaincfg "GWatch/internal/domain/config"
	"GWatch/internal/domain/notifier"

	"github.com/youxihu/dingtalk/dingtalk"
)

// DingTalkNotifier implements Notifier using youxihu/dingtalk.
type DingTalkNotifier struct{ provider domaincfg.Provider }

func NewDingTalkNotifier(p domaincfg.Provider) notifier.Notifier {
	return &DingTalkNotifier{provider: p}
}

func (d *DingTalkNotifier) Send(title string, markdown string) error {
	cfg := d.provider.GetConfig()
	if cfg == nil {
		return nil
	}
	return dingtalk.SendDingDingNotification(cfg.DingTalk.WebhookURL, cfg.DingTalk.Secret, title, markdown, cfg.DingTalk.AtMobiles, false)
}
