package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/parkr/gossip/database"
	"github.com/parkr/gossip/template"
)

func ensureLeadingHash(room string) string {
	if strings.HasPrefix(room, "#") {
		return room
	}
	return "#" + room
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	newReq, _ := http.NewRequest(r.Method, "/room/%23jekyll", nil)
	h.LatestMessagesByRoom(w, newReq)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")
	if query == "" {
		http.Error(w, "no search term given", http.StatusBadRequest)
		return
	}

	messages, err := h.DB.ListByFuzzyMessage(query)
	if err != nil {
		fmt.Fprintf(w, "\n\ncouldn't fetch messages: %+v", err)
		http.Error(w, "couldn't fetch messages", http.StatusInternalServerError)
		return
	}

	messagesGroupedByRoom := map[string][]database.Message{}
	for _, message := range messages {
		if _, ok := messagesGroupedByRoom[message.Room]; !ok {
			messagesGroupedByRoom[message.Room] = []database.Message{}
		}
		messagesGroupedByRoom[message.Room] = append(messagesGroupedByRoom[message.Room], message)
	}
	data := &template.SearchTemplateData{
		Results: messagesGroupedByRoom,
		Total:   len(messages),
		Rooms:   h.AllRooms(),
		Query:   query,
	}
	if err := template.SearchTemplate.Execute(w, data); err != nil {
		fmt.Fprintf(w, "\n\n%+v", err)
	}
}

func (h *Handler) LatestMessagesByRoom(w http.ResponseWriter, r *http.Request) {
	unescapedURLPath, err := url.PathUnescape(r.URL.Path)
	if err != nil {
		logForReq(r, fmt.Sprintf("Couldn't unescape URL.Path '%s': %+v", r.URL.Path, err))
		unescapedURLPath = r.URL.Path
	}
	room := ensureLeadingHash(strings.TrimPrefix(unescapedURLPath, "/room/"))

	messages, err := h.DB.LatestMessagesByRoom(room, 20)
	if err == sql.ErrNoRows || len(messages) == 0 {
		http.Error(w, "no results for "+room, http.StatusNotFound)
		return
	}
	if err != nil {
		fmt.Fprintf(w, "\n\ncouldn't fetch messages: %+v", err)
		http.Error(w, "couldn't fetch messages", http.StatusInternalServerError)
		return
	}
	data := &template.ListTemplateData{
		Messages:    messages,
		Rooms:       h.AllRooms(),
		CurrentRoom: room,
	}
	if err := template.ListTemplate.Execute(w, data); err != nil {
		fmt.Fprintf(w, "\n\n%+v", err)
	}
}

func (h *Handler) LatestMessagesByAuthor(w http.ResponseWriter, r *http.Request) {
	author := strings.TrimPrefix(r.URL.Path, "/messages/by/")
	messages, err := h.DB.LatestMessagesByAuthor(author, 20)
	if err == sql.ErrNoRows || len(messages) == 0 {
		http.Error(w, "no results for "+author, http.StatusNotFound)
		return
	}
	if err != nil {
		fmt.Fprintf(w, "\n\ncouldn't fetch messages: %+v", err)
		http.Error(w, "couldn't fetch messages", http.StatusInternalServerError)
		return
	}
	data := &template.ListTemplateData{
		Messages:      messages,
		Rooms:         h.AllRooms(),
		CurrentAuthor: author,
	}
	if err := template.ListTemplate.Execute(w, data); err != nil {
		fmt.Fprintf(w, "\n\n%+v", err)
	}
}

func (h *Handler) MessageContext(w http.ResponseWriter, r *http.Request) {
	limit := 5
	messageIDStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/messages/"), "/context")
	if messageIDStr == "" {
		http.Error(w, "no message id given", http.StatusBadRequest)
		return
	}
	messageID, err := strconv.Atoi(messageIDStr)
	if err != nil || messageID == 0 {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}

	message, err := h.DB.Find(messageID)
	if err == sql.ErrNoRows {
		http.Error(w, "no message with id "+messageIDStr, http.StatusNotFound)
		return
	}
	if err != nil {
		fmt.Fprintf(w, "\n\ncouldn't fetch message: %+v", err)
		http.Error(w, "couldn't fetch message", http.StatusInternalServerError)
		return
	}
	priorMessages, err := h.DB.PriorMessages(message.Room, message.At, limit)
	if err != nil && err != sql.ErrNoRows {
		fmt.Fprintf(w, "\n\ncouldn't fetch prior messages: %+v", err)
		http.Error(w, "couldn't fetch prior messages", http.StatusInternalServerError)
		return
	}
	subsequentMessages, err := h.DB.SubsequentMessages(message.Room, message.At, limit)
	if err != nil && err != sql.ErrNoRows {
		fmt.Fprintf(w, "\n\ncouldn't fetch subsequent messages: %+v", err)
		http.Error(w, "couldn't fetch subsequent messages", http.StatusInternalServerError)
		return
	}

	data := &template.ShowTemplateData{
		PriorMessages:      priorMessages,
		Message:            *message,
		SubsequentMessages: subsequentMessages,
		Rooms:              h.AllRooms(),
		CurrentRoom:        message.Room,
	}
	if err := template.ShowTemplate.Execute(w, data); err != nil {
		fmt.Fprintf(w, "\n\n%+v", err)
	}
}
