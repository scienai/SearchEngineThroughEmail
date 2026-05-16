package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sync"

	//"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/imapclient"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/text/textcore"
)

var searchid int
var msg *core.Text

func createNewTab(tabs *core.Tabs) {
	tab1frm, tab1 := tabs.NewTab("Search" + strconv.Itoa(searchid))
	searchid += 1
	tab1.SetIcon(icons.List)
	searchbarfrm := core.NewFrameRow(tab1frm)
	email := core.NewTextField(searchbarfrm)
	email.SetPlaceholder("email@domain.com")
	emailPassword := core.NewTextField(searchbarfrm)
	emailPassword.SetTypePassword()
	emailPassword.SetPlaceholder("Email password")
	searchText := core.NewTextField(searchbarfrm)
	imap := core.NewTextField(searchbarfrm)
	imap.SetPlaceholder("imap.domain.com:993")
	smtp := core.NewTextField(searchbarfrm)
	smtp.SetPlaceholder("smtp.domain.com:25")
	searchText.SetPlaceholder("Search text")
	serarchbtn := core.NewButton(searchbarfrm).SetIcon(icons.Search)
	serarchbtn.SetTooltip("Search text use your email send it to APP email and wait reply result.")
	edit1 := textcore.NewEditor(tab1frm)
	cfgctt, _ := ioutil.ReadFile("useremail.cfg")
	cfglis := strings.Split(string(cfgctt), "\t")
	if len(cfglis) >= 4 {
		email.SetText(cfglis[0])
		emailPassword.SetText(cfglis[1])
		imap.SetText(cfglis[2])
		smtp.SetText(cfglis[3])
	}
	serarchbtn.OnClick(func(e events.Event) {
		ioutil.WriteFile("useremail.cfg", []byte(email.Text()+"\t"+emailPassword.Text()+"\t"+imap.Text()+"\t"+smtp.Text()), 0666)
		stxt := strings.Trim(searchText.Text(), " \r\n\t")
		if stxt != "" {
			go func() {
				resulttext, err := UserSendSearchTextAndReceiveResult(email.Text(), emailPassword.Text(), imap.Text(), searchText.Text())
				if err == nil {
					resulttext = strings.Trim(resulttext, " \r\n\t")
					resulttext = strings.Replace(resulttext, "\\n", "\n", -1)
					edit1.Lines.SetString(resulttext)
				} else {
					edit1.Lines.SetString(err.Error())
				}
			}()
			msg.SetText("Send search text")
			UserSendEmailSeaarchText(email.Text(), emailPassword.Text(), smtp.Text(), "suirosu6@163.com", searchText.Text())
		}
	})

}

func UserSendEmailSeaarchText(emailAddr, emailPwd, smtpServer string, sendto string, subject string) {
	showname := emailAddr[:strings.Index(emailAddr, "@")]
	auth := smtp.PlainAuth("", emailAddr, emailPwd, strings.Split(smtpServer, ":")[0])
	msg := "To: " + sendto + "\r\nFrom: " + showname + "<" + emailAddr + ">\r\nSubject: " + subject + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + "user send search engine throgh email search text."
	senderr := smtp.SendMail(smtpServer, auth, emailAddr, []string{sendto}, []byte(msg))
	if senderr != nil {
		log.Println("user send search text email error:", senderr)
	} else {
		log.Println("user send search text subject", "\""+subject+"\"", "send email OK.")
	}
}

func UserSendSearchTextAndReceiveResult(emailAddr, emailPwd, imapServer, subject string) (result string, er error) {
	// 连接 IMAP 服务器（使用 TLS）
	c, err := imapclient.DialTLS(imapServer, nil)
	if err != nil {
		return "", fmt.Errorf("connect IMAP error: %v", err)
	}
	defer c.Logout()

	// 登录
	fmt.Println("emailAddr", emailAddr, "emailPwd", emailPwd)
	lgcmd := c.Login(emailAddr, emailPwd)
	lgerr := lgcmd.Wait()
	if lgerr != nil {
		return "", fmt.Errorf("login error: %v", lgerr)
	}

	idcmd := c.ID(&imap.IDData{
		Name:    "setebox",
		Version: "1.0",
	})
	idcmd.Wait()

	//imap服务器支持什么
	fmt.Println("imap support:", c.Caps())
	//
	inboxcmd := c.Select("INBOX", &imap.SelectOptions{ReadOnly: true, CondStore: false})
	mbox, inboxer := inboxcmd.Wait()
	if inboxer != nil {
		return "", fmt.Errorf("INBOX error: %v", inboxer)
	}
	fmt.Println("mbox NumMessages", mbox.NumMessages)

	waitsec := 5
	for {
		closed := false
		var idlewaiter error
		var idleclose sync.Mutex
		var t15mchan *time.Ticker
		idlecmd, ideler := c.Idle()
		if ideler != nil {
			//可能是服务器不支持Idle, Idle需要IMAP4v2版本才支持
			time.Sleep(time.Duration(waitsec) * time.Second)
			goto sleepCheck
			//return fmt.Errorf("Idle失败: %v", ideler)
		}
		t15mchan = time.NewTicker(15 * 60 * time.Second)
		go func() {
			<-t15mchan.C
			idleclose.Lock()
			if closed == false {
				idlecmd.Close()
				closed = true
			}
			idleclose.Unlock()
		}()
		idlewaiter = idlecmd.Wait()
		if idlewaiter != nil {
			return "", fmt.Errorf("Idle Wait error: %v", idlewaiter)
		}
		idleclose.Lock()
		if closed == false {
			fmt.Println("have new email close idle result:", idlecmd.Close())
			closed = true
		}
		idleclose.Unlock()
		t15mchan.Stop()
	sleepCheck:
		waitsec = 30
		// 获取最新一封邮件
		seqSet := imap.SeqSetNum(1, 2, 3, 4, 5) // 获取最后一封
		var bdsec = []*imap.FetchItemBodySection{{
			Specifier: imap.PartSpecifierText,
			Peek:      true,
		}}
		// 获取邮件头和正文
		fcmd := c.Fetch(seqSet, &imap.FetchOptions{
			Envelope: true,
			//Flags:        true,
			//InternalDate: true,
			//RFC822Size:  true,
			UID:         true,
			BodySection: bdsec,
			//BinarySection     []*FetchItemBinarySection     // requires IMAP4rev2 or BINARY
			//BinarySectionSize []*FetchItemBinarySectionSize // requires IMAP4rev2 or BINARY
			//ModSeq: false, //            bool                          // requires CONDSTORE
			//ChangedSince uint64 // requires CONDSTORE
		})
		fcndn := fcmd.Next()
		allrl := []string{}
		for fcndn != nil {
			fid := fcndn.Next()
			for fid != nil {
				col, cole := fcndn.Collect()
				if cole == nil {
					fmt.Println("col.Envelope", col.Envelope)
					if strings.HasPrefix(col.Envelope.Subject, "SETE-Reply-"+subject) {
						fmt.Println("pass subject", col.Envelope.Subject)
						searchresult := string(col.FindBodySection(bdsec[0]))
						msg.SetText("email receive search text result complete.")
						allrl = append(allrl, searchresult)
					}
				} else {
					fmt.Println("imapclient.FetchMessageData Collect error:", cole)
					break
				}
				fid = fcndn.Next()
			}
			fcndn = fcmd.Next()
		}
		if len(allrl) > 0 {
			return allrl[len(allrl)-1], nil
		}
	}
	return "", errors.New("user receive email quit.")
}

func main() {
	if len(os.Args) > 1 {
		help := `Search Engine Through Email Help:
command:
	help (show help)
	server server_reply_email@domail.com imap_server:port smtp_server:port (run as search engin through email server, default parse key_value.csv comma key value table for search.)
`
		switch os.Args[1] {
		case "help":
			fmt.Println(help)
		case "server":
			fmt.Println("Please input server reply search email password:")
			pwd, _ := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println("run server return:", RunSearchEngineThrouEmailSever(os.Args[2], string(pwd), os.Args[3], os.Args[4]))
			return
		default:
			fmt.Println("error parameter:", os.Args)
			fmt.Println(help)
			return
		}
	}

	mainwin := core.NewBody()
	maincol := core.NewFrameCol(mainwin)
	// memubarfrm := core.NewFrameRow(maincol)
	// menu := func(m *core.Scene) {
	// 	m1 := core.NewButton(m).SetText("Exit")
	// 	m1.SetTooltip("Exit this program")
	// 	m1.OnClick(func(e events.Event) {
	// 		mainwin.Close()
	// 	})
	// 	m2 := core.NewButton(m).SetText("About")
	// 	m2.SetTooltip("About this program")
	// 	m2.OnClick(func(e events.Event) {
	// 		dlg := core.NewBody("About Search Engine Through Email")
	// 		core.NewText(dlg).SetText("Search Engine Through Email 0.1 2026-05-14 by Luevzranr")
	// 		dlg.AddOKOnly()
	// 		dlg.RunDialog(m2)
	// 	})
	// }
	// core.NewButton(memubarfrm).SetIcon(icons.Menu).SetMenu(menu)

	tabs := core.NewTabs(maincol)
	tabs.SetNewTabButton(true)
	tabs.NewTabButtonOnClick = func(e events.Event) {
		createNewTab(tabs)
		tabs.SelectTabIndex(tabs.NumTabs() - 1)
	}
	createNewTab(tabs)
	msg = core.NewText(maincol)
	msg.SetText("Search Engine Through Email 0.1 2026-05-14 by Luevzranr")
	mainwin.RunMainWindow()
}

func RunSearchEngineThrouEmailSever(emailAddr, emailPwd, imapServer, smtpServer string) error {
	// 连接 IMAP 服务器（使用 TLS）
	c, err := imapclient.DialTLS(imapServer, nil)
	if err != nil {
		return fmt.Errorf("connect IMAP error: %v", err)
	}
	defer c.Logout()

	// 登录
	lgcmd := c.Login(emailAddr, emailPwd)
	lgerr := lgcmd.Wait()
	if lgerr != nil {
		return fmt.Errorf("login error: %v", lgerr)
	}

	idcmd := c.ID(&imap.IDData{
		Name:    "setebox",
		Version: "1.0",
	})
	idcmd.Wait()

	//imap服务器支持什么
	fmt.Println("imap support:", c.Caps())
	//
	inboxcmd := c.Select("INBOX", &imap.SelectOptions{ReadOnly: true, CondStore: false})
	mbox, inboxer := inboxcmd.Wait()
	if inboxer != nil {
		return fmt.Errorf("INBOX error: %v", inboxer)
	}
	fmt.Println("mbox NumMessages", mbox.NumMessages)

	kvctt, _ := ioutil.ReadFile("key_value.csv")
	lis := strings.Split(string(kvctt), "\n")
	kvctt = nil
	keyvalmap := make(map[string]string)
	for _, li := range lis {
		ci := strings.Index(li, ",")
		if ci != -1 {
			keyvalmap[li[:ci]] = li[ci+1:]
		}
	}
	lis = nil
	uidctt, _ := ioutil.ReadFile("email_uid_sended.log")
	uidlis := strings.Split(string(uidctt), "\n")
	sendeduid := make(map[string]bool)
	for _, li := range uidlis {
		msgid := strings.Split(strings.Trim(li, " \r\n\t"), "\t")[0]
		if len(msgid) > 0 {
			sendeduid[msgid] = true
		}
	}
	uidf, _ := os.OpenFile("email_uid_sended.log", os.O_CREATE|os.O_RDWR, 0666)
	waitsec := 0
	for {
		closed := false
		var idlewaiter error
		var idleclose sync.Mutex
		var t15mchan *time.Ticker
		idlecmd, ideler := c.Idle()
		if ideler != nil {
			//可能是服务器不支持Idle, Idle需要IMAP4v2版本才支持
			time.Sleep(time.Duration(waitsec) * time.Second)
			goto sleepCheck
			//return fmt.Errorf("Idle失败: %v", ideler)
		}
		t15mchan = time.NewTicker(15 * 60 * time.Second)
		go func() {
			<-t15mchan.C
			idleclose.Lock()
			if closed == false {
				idlecmd.Close()
				closed = true
			}
			idleclose.Unlock()
		}()
		idlewaiter = idlecmd.Wait()
		if idlewaiter != nil {
			return fmt.Errorf("Idle Wait error: %v", idlewaiter)
		}
		idleclose.Lock()
		if closed == false {
			fmt.Println("have new email close idle result:", idlecmd.Close())
			closed = true
		}
		idleclose.Unlock()
		t15mchan.Stop()
	sleepCheck:
		waitsec = 60
		// 获取最新一封邮件
		seqSet := imap.SeqSetNum(1, 2, 3, 4, 5, 6, 7, 8, 9, 10) // 获取最后一封
		var bdsec = []*imap.FetchItemBodySection{{
			Specifier: imap.PartSpecifierText,
			Peek:      true,
		}}
		// 获取邮件头和正文
		fcmd := c.Fetch(seqSet, &imap.FetchOptions{
			Envelope: true,
			//Flags:        true,
			//InternalDate: true,
			//RFC822Size:  true,
			UID:         true,
			BodySection: bdsec,
			//BinarySection     []*FetchItemBinarySection     // requires IMAP4rev2 or BINARY
			//BinarySectionSize []*FetchItemBinarySectionSize // requires IMAP4rev2 or BINARY
			//ModSeq: false, //            bool                          // requires CONDSTORE
			//ChangedSince uint64 // requires CONDSTORE
		})
		fcndn := fcmd.Next()
		for fcndn != nil {
			fid := fcndn.Next()
			for fid != nil {
				col, cole := fcndn.Collect()
				if cole == nil {
					fmt.Println("col.Envelope", col.Envelope)
					if strings.HasPrefix(col.Envelope.Subject, "SETE-") {
						fmt.Println("pass subject", col.Envelope.Subject, "messageID", col.Envelope.MessageID)
						_, uidok := sendeduid[col.Envelope.MessageID]
						if uidok == false {
							sendeduid[col.Envelope.MessageID] = true
							uidf.Write([]byte(col.Envelope.MessageID + "\t" + col.Envelope.Subject + "\n"))
							fmt.Println("pass msgid subject", col.Envelope.Subject)
							searchtext := col.Envelope.Subject[len("SETE-"):]
							fmt.Println("search text:", searchtext)
							searchtextre, searchtexterr := regexp.Compile(searchtext)
							if searchtexterr == nil {
								resulttext := []byte{}
								for key, val := range keyvalmap {
									if searchtextre.MatchString(key) {
										resulttext = append(resulttext, []byte(key+":"+val+"\n")...)
									}
								}
								if len(resulttext) == 0 {
									resulttext = []byte("Not found")
								}
								fmt.Println("reply send to:", col.Envelope.Sender[0].Addr())
								ServerReplySearchSendEmail(emailAddr, emailPwd, smtpServer, col.Envelope.Sender[0].Addr(), searchtext, string(resulttext))
							}
						}
					}
				} else {
					fmt.Println("imapclient.FetchMessageData Collect error:", cole)
					break
				}
				fid = fcndn.Next()
			}
			fcndn = fcmd.Next()
		}
	}
	uidf.Close()
	return errors.New("Search enginer through email server quit.")
}

func ServerReplySearchSendEmail(emailAddr, emailPwd, smtpServer string, sendto string, subject, bodytext string) {
	bodytext = strings.Replace(bodytext, "\r\n", "\n", -1)
	bodytext = strings.Replace(bodytext, "\\n", "\n", -1)
	showname := emailAddr[:strings.Index(emailAddr, "@")]
	auth := smtp.PlainAuth("", emailAddr, emailPwd, strings.Split(smtpServer, ":")[0])
	msg := "To: " + sendto + "\r\nFrom: " + showname + "<" + emailAddr + ">\r\nSubject: " + "SETE-Reply-" + subject + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + bodytext
	senderr := smtp.SendMail(smtpServer, auth, emailAddr, []string{sendto}, []byte(msg))
	if senderr != nil {
		log.Println("sever replay search send email error:", senderr)
	} else {
		log.Println("sever replay search subject", "\""+subject+"\"", "send email OK.")
	}
}
