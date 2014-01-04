/*
 *  TVPN: A Peer-to-Peer VPN solution for traversing NAT firewalls
 *  Copyright (C) 2013  Joshua Chase <jcjoshuachase@gmail.com>
 *
 *  This program is free software; you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation; either version 2 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License along
 *  with this program; if not, write to the Free Software Foundation, Inc.,
 *  51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
*/

package irc

import (
	"sync"
	"fmt"
	"github.com/Pursuit92/irc"
	"github.com/Pursuit92/tvpn"
)

func SetLogLevel(n int) {
	irc.SetLogLevel(n)
}

type IRCBackend struct {
	Nick,Chan,Server string
	Conn        *irc.Conn
	Messages    chan irc.Command
	Status      chan irc.Command
}

func (i *IRCBackend) Configure(conf tvpn.SigConfig) {
	i.Nick = conf["Name"]
	i.Chan = conf["Group"]
	i.Server = conf["Server"]

	if i.Conn != nil {
		// cleanup old connection stuff
	}

	err := i.Connect()

	if err != nil {
		panic(err)
	}

}

func (i *IRCBackend) Connect() error {

	conn, err := irc.DialIRC(i.Server, []string{i.Nick}, i.Nick, i.Nick)
	if err != nil {
		return err
	}
	_, err = conn.Register()
	if err != nil {
		return err
	}

	chann, err := conn.Join(i.Chan)
	if err != nil {
		return err
	}

	joinpart, _ := irc.Expect(chann, irc.Command{"", "(JOIN)|(PART)", []string{}})
	quit, _ := irc.Expect(conn, irc.Command{"", "QUIT", []string{}})
	status := make(chan irc.Command)
	// Combine joinpart and quit into one channel
	go func() {
		for {
			select {
			case msg := <-joinpart.Chan:
				status <- msg
			case msg := <-quit.Chan:
				status <- msg
			}
		}
	}()

	msgs, err := irc.Expect(conn, irc.Command{"", "PRIVMSG", []string{i.Nick,".*"}})
	if err != nil {
		return err
	}

	//users := chann.GetUsers()
	//go makeJoin(users, status)

	i.Conn = conn
	i.Messages = msgs.Chan
	i.Status = status

	return nil
}

func makeJoin(users map[string]irc.IRCUser, status chan<- irc.Command) {
	for _, v := range users {
		status <- irc.Command{v.String(), irc.Join, []string{}}
	}
}

func (b IRCBackend) RecvMessage() tvpn.Message {
	for {
		select {
		case input := <-b.Messages:
			ircMsg := input.Message()
			msg, err := tvpn.ParseMessage(input.Params[len(input.Params)-1])
			if err == nil {
				msg.From = ircMsg.Nick
				return *msg
			} else {
				fmt.Printf("Failed to parse message!")
			}
		case input := <-b.Status:
			switch input.Command {
			case "QUIT", "PART":
				return tvpn.Message{From: input.Message().Nick, Type: tvpn.Quit}
			case "JOIN":
				if input.Message().Nick != b.Conn.Nick {
					return tvpn.Message{From: input.Message().Nick, Type: tvpn.Join}
				}
			}
		}

	}
}

func (b IRCBackend) SendMessage(mes tvpn.Message) error {
	return b.Conn.Send(irc.Command{b.Conn.Nick, irc.Privmsg, []string{mes.To, mes.String()}})
}

func combine(inputs []<-chan irc.Command, output chan<- irc.Command) {
	var group sync.WaitGroup
	for i := range inputs {
		group.Add(1)
		go func(input <-chan irc.Command) {
			for val := range input {
				output <- val
			}
			group.Done()
		} (inputs[i])
	}
	go func() {
		group.Wait()
		close(output)
	} ()
}
