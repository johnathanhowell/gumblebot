package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/layeh/gumble/gumble"
	"html/template"
	"io/ioutil"
	"strings"
)

const (
	GumblebotRoot      = 2
	GumblebotModerator = 1
	GumblebotUser      = 0

	permissiondenied = "I'm sorry dave, I can't do that."
	whoistemplate = `
		<br></br><b> Whois {{ .Name }} </b>
		<ul>
			<li>{{.AccessLevel}} </li>
		<ul>`
)

type AdminUser struct {
	UserName    string
	AccessLevel uint
}

type MumbleAdmin struct {
	Users map[string]AdminUser
}
type WhoisContext struct {
	Name        string
	AccessLevel string
}

func (m *MumbleAdmin) LoadAdminData(datapath string) {
	m.Users = make(map[string]AdminUser)
	iobuffer, err := ioutil.ReadFile(datapath)
	if err != nil {
		fmt.Println(err)
		return
	}
	buffer := bytes.NewBuffer(iobuffer)
	dec := gob.NewDecoder(buffer)
	err = dec.Decode(&m.Users)
	if err != nil {
		panic(err)
	}
}
func (m *MumbleAdmin) SaveAdminData(datapath string) {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(m.Users)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(datapath, buffer.Bytes(), 0755)
	if err != nil {
		panic(err)
	}
}
func (m *MumbleAdmin) RegisterUser(user string, accesslevel uint) {
	m.Users[user] = AdminUser{UserName: user, AccessLevel: accesslevel}
}
func search_mumble_users_substring(target string, client *gumble.Client) *gumble.User {
	for _, user := range client.Users {
		if strings.Index(strings.ToLower(user.Name), strings.ToLower(target)) == 0 {
			return user
		}
	}
	return nil
}
func (m *MumbleAdmin) Move (sender *gumble.User, client *gumble.Client, channelsubstring string, users []string) {
	if user, ok := m.Users[sender.Name]; ok {
		if user.AccessLevel < GumblebotModerator {
			SendMumbleMessage(permissiondenied, client, client.Self.Channel)
			return
		}
		var targetChannel *gumble.Channel
		for _, channel := range client.Channels {
			channelname := strings.ToLower(channel.Name)
			if strings.Index(channelname, strings.ToLower(channelsubstring)) == 0 {
				targetChannel = channel
			}
		}
		if targetChannel == nil {
			nochanerr := fmt.Sprintf("No such channel: %s", channelsubstring)
			SendMumbleMessage(nochanerr, client, client.Self.Channel)
			return
		}
		for _, targetusersubstring := range users {
			targetuser := search_mumble_users_substring(targetusersubstring, client)
			if targetuser == nil {
				nousererr := fmt.Sprintf("No such user: %s", targetusersubstring)
				SendMumbleMessage(nousererr, client, client.Self.Channel)
				return
			}
			targetuser.Move(targetChannel)
		}
	}
}
func (m *MumbleAdmin) Whois(sender *gumble.User, targetusername string, client *gumble.Client) {
	if user, ok := m.Users[sender.Name]; ok {
		if user.AccessLevel >= GumblebotUser {
			targetuser := search_mumble_users_substring(targetusername, client)
			if targetuser == nil {
				// no such user, return!
				return
			}
			var accessLevel string
			if targetadmin, ok := m.Users[targetuser.Name]; ok {
				switch targetadmin.AccessLevel {
				case GumblebotRoot:
					accessLevel = "Root Administrator"
				case GumblebotModerator:
					accessLevel = "Mumble Moderator"
				case GumblebotUser:
					accessLevel = "Mumble User"
				}
			} else {
				accessLevel = "Mumble User"
			}

			var buffer bytes.Buffer
			template, err := template.New("whois").Parse(whoistemplate)
			if err != nil {
				panic(err)
			}
			err = template.Execute(&buffer, WhoisContext{targetuser.Name, accessLevel})

			if err != nil {
				panic(err)
			}
			message := gumble.TextMessage{
				Channels: []*gumble.Channel{
					client.Self.Channel,
				},
				Message: buffer.String(),
			}
			client.Send(&message)
		}
	}
}
