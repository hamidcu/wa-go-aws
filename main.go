package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"strings"
)

type MyClient struct {
	WAClient       *whatsmeow.Client
	eventHandlerID uint32
}

func (mycli *MyClient) register() {
	mycli.eventHandlerID = mycli.WAClient.AddEventHandler(mycli.eventHandler)
}

// Message received by client to business account
func (mycli *MyClient) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		newMessage := v.Message
		userNumber := v.Info.Sender.User

		is_group := 1
		if v.Info.Chat != v.Info.Sender {
			is_group = 1
		} else {
			is_group = 0
		}

		msg := newMessage.GetConversation()

		if msg != "" {

			userJid := types.NewJID(v.Info.Sender.User, types.DefaultUserServer)

			//****************  mark the client as online **********
			mycli.WAClient.SendPresence(types.PresenceAvailable)

			// ************ Send Message Readed Recipet to Whatsapp ************//
			mycli.WAClient.MarkRead([]string{v.Info.ID}, time.Now(), v.Info.Chat, v.Info.Sender)
			userInfoMap, err := mycli.WAClient.GetUserInfo([]types.JID{v.Info.Sender})

			if err != nil {
				// Handle getting user info error
			}
			// ******************** U S E R --- I N F O *******************//
			// Loop through the user info map and print the results
			for jid, userInfo := range userInfoMap {
				fmt.Println("JID:", jid)
				fmt.Println("Avatar URL:", userInfo.PictureID)
				fmt.Println("Status:", userInfo.Status)
				// fmt.Println("Verified Business Name:", userInfo.VerifiedName.Details)
				fmt.Println("Device List:", userInfo.Devices)
				fmt.Println()
			}
			fmt.Println("jbr Id", userJid)
			return
			//************************* User T Y P I N G or Recording -- ETC ************//
			mycli.WAClient.SendChatPresence(userJid, types.ChatPresenceComposing, types.ChatPresenceMediaText)

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			_ = writer.WriteField("message", msg)

			_ = writer.WriteField("userNumber", userNumber)

			_ = writer.WriteField("is_group", strconv.Itoa(is_group))
			// Don't forget to close the multipart writer
			writer.Close()
			url := "https://www.naharmagra.com/api/api-proceess" // your server url or route where you want to recieve the data
			// Make a POST request to the URL
			req, err := http.NewRequest("POST", url, body)
			if err != nil {
				return
			}

			req.Header.Set("Content-Type", writer.FormDataContentType())

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			// Read the response
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			newMsg := buf.String()

			if newMsg == "" {
				// Do nothing if newMsg is empty
				mycli.WAClient.SendChatPresence(userJid, types.ChatPresencePaused, types.ChatPresenceMediaText)
				return
			}

			// Count the words
			// wordCount := len(splitWords(newMsg))

			// Generate a random typing speed between 25 and 30 words per minute
			// typingSpeed := rand.Intn(6) + 25 // Generates a random number between 25 and 30 (inclusive)

			// Calculate the estimated typing time in seconds and round to the nearest second
			// estimatedTime := (float64(wordCount) / float64(typingSpeed)) * 60
			// roundedTime := int(estimatedTime + 0.5) // Round to the nearest second

			// Introduce a delay in the response equal to the roundedTime
			// time.Sleep(time.Duration(typingSpeed) * time.Second)

			response := &waProto.Message{Conversation: proto.String(string(newMsg))}
			mycli.WAClient.SendMessage(context.Background(), userJid, response)
			fmt.Println("Response:", string(newMsg))

			//****************  mark the client as offline **********
			mycli.WAClient.SendPresence(types.PresenceUnavailable)

			// Open the file
			// file, err := os.Open("whatsapp_numbers.txt")
			// if err != nil {
			//      fmt.Println("Error opening file:", err)
			//      return
			// }
			// defer file.Close()

			// // Read phone numbers from the file
			// var recipients []string
			// scanner := bufio.NewScanner(file)
			// for scanner.Scan() {
			//      phoneNumber := strings.TrimSpace(scanner.Text())
			//      if phoneNumber != "" {
			//              recipients = append(recipients, phoneNumber)
			//      }

			//      // Limit the number of recipients to 50
			//      if len(recipients) >= 3 {
			//              break
			//      }
			// }

			// if err := scanner.Err(); err != nil {
			//      fmt.Println("Error reading file:", err)
			//      return
			// }

			// // Send the reply to each recipient
			// for _, recipient := range recipients {
			//      userJID := types.NewJID(recipient, types.DefaultUserServer)
			//      mycli.WAClient.SendMessage(context.Background(), userJID, response)
			// }
			// // Read and print the response
			// responseBody, err := io.ReadAll(resp.Body)
			// if err != nil {
			//      return
			// }

			// fmt.Println("Response:", string(responseBody))

		}

		// Perform cleanup of old files
		// errs := cleanupDataDirectory()
		// if errs != nil {
		//      fmt.Println("Error cleaning up data directory:", errs)
		// }

	}
}

// Helper function to split words
func splitWords(text string) []string {
	// Implement your own logic to split words
	// For simplicity, this example uses space as the word delimiter
	return strings.Fields(text)
}

func cleanupDataDirectory() error {
	// Define the data directory path
	dataDir := "./data/"

	// Set the maximum age for file cleanup (5 minutes)
	maxAge := 1 * time.Minute
	files, err := os.ReadDir(dataDir)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			fmt.Printf("Error getting file info: %v\n", err)
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			filePath := filepath.Join(dataDir, file.Name())
			err := os.Remove(filePath)
			if err != nil {
				fmt.Printf("Error deleting file %s: %v\n", filePath, err)
			} else {
				fmt.Printf("Deleted file: %s\n", filePath)
			}
		}
	}

	return nil
}

func uploadData(filename string, userNumber string, caption string, filetype string, message string) error {

	// Create a new buffer to store the multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fmt.Println("Response mesg here:", message)

	if filename != "" {
		// Open the file
		file, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("error opening file: %v", err)
		}
		defer file.Close()

		// Create a form field for the file
		part, err := writer.CreateFormFile("data", filename)
		if err != nil {
			return fmt.Errorf("error creating form file: %v", err)
		}

		// Copy the file content to the form field
		_, err = io.Copy(part, file)
		if err != nil {
			return fmt.Errorf("error copying file content: %v", err)
		}
	}
	if message != "" {
		_ = writer.WriteField("message", message)
	} else {

		// Add caption as a form field if not empty
		if caption != "" {
			_ = writer.WriteField("caption", caption)
		}

		if filetype != "" {
			_ = writer.WriteField("filetype", filetype)
		}
	}

	// Add user number as a form field if not empty
	if userNumber != "" {
		_ = writer.WriteField("userNumber", userNumber)
	}

	// Don't forget to close the multipart writer
	writer.Close()
	url := "https://whatinthmd.000webhostapp.com/wp-json/wa-app-chat/v1/get-or-post" // your server url or route where you want to recieve the data
	// Make a POST request to the URL
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read and print the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	fmt.Println("Response:", string(responseBody))

	return nil
}

// message received from server by bussiness account
func (mycli *MyClient) handler(w http.ResponseWriter, r *http.Request) {
	// Check if the request method is POST
	if r.Method == http.MethodPost {

		// Close the request body to prevent resource leaks
		defer r.Body.Close()

		userNumber := r.FormValue("number")
		if userNumber == "" {
			http.Error(w, "user number empty", http.StatusUnprocessableEntity)
			return
		}
		captionTxt := ""

		passcode := r.FormValue("passcode")
		if passcode == "" {
			http.Error(w, "passcode is empty", http.StatusUnprocessableEntity)
			return
		}

		captionTxtServer := r.FormValue("caption")
		if captionTxtServer != "" {
			captionTxt = captionTxtServer
		}

		expectedPasscode := "123456"
		if passcode != expectedPasscode {
			http.Error(w, "Invalid passcode", http.StatusUnauthorized)
			return
		}
		// with country code Without + (symbol)
		userJid := types.NewJID(userNumber, types.DefaultUserServer)
		// Retrieve the uploaded file
		file, handler, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			// Determine the file type
			fileType := "text"
			// if strings.HasPrefix(handler.Header.Get("Content-Type"), "image/") {
			//      fileType = "image"
			// } else if strings.HasPrefix(handler.Header.Get("Content-Type"), "video/") {
			//      fileType = "video"
			// } else if strings.HasPrefix(handler.Header.Get("Content-Type"), "audio/") {
			//      fileType = "audio"

			// }

			// fmt.Println("the file type: ", fileType)
			contentType := handler.Header.Get("Content-Type")
			if strings.HasPrefix(contentType, "image/") {
				fileType = "image"
			} else if strings.HasPrefix(contentType, "video/") {
				fileType = "video"
			} else if strings.HasPrefix(contentType, "audio/") {
				fileType = "audio"
			} else if strings.HasPrefix(contentType, "application/") {
				fileType = "doc"
			}

			fmt.Println("The file type:", fileType)
			dataDir := "./data/"
			// Create a new file to save the uploaded content
			newFilename := fmt.Sprintf("%s%s.%s", dataDir, fileType, strings.Split(handler.Filename, ".")[1])
			newFile, err := os.Create(newFilename)
			if err != nil {
				http.Error(w, "Unable to create file", http.StatusInternalServerError)
				return
			}
			defer newFile.Close()

			// Copy the uploaded content to the new file
			_, err = io.Copy(newFile, file)
			if err != nil {
				http.Error(w, "Error copying file content", http.StatusInternalServerError)
				return
			}

			if fileType == "audio" {
				// Convert the file path to a byte slice
				audioBytes, err := os.ReadFile(newFilename)
				if err != nil {
					panic(err)
				}
				// Upload the audio message to WhatsApp
				uploadedAudio, err := mycli.WAClient.Upload(context.Background(), audioBytes, whatsmeow.MediaAudio)
				if err != nil {
					panic(err)
				}

				// Create a message object with the audio message
				message := &waProto.Message{
					AudioMessage: &waProto.AudioMessage{
						URL:           proto.String(uploadedAudio.URL),
						PTT:           proto.Bool(true),
						DirectPath:    proto.String(uploadedAudio.DirectPath),
						MediaKey:      uploadedAudio.MediaKey,
						Mimetype:      proto.String("audio/ogg; codecs=opus"),
						FileEncSHA256: uploadedAudio.FileEncSHA256,
						FileSHA256:    uploadedAudio.FileSHA256,
						FileLength:    proto.Uint64(uint64(len(newFilename))),
					},
				}

				success, err := mycli.WAClient.SendMessage(context.Background(), userJid, message)
				if err != nil {
					panic(err)
				}
				fmt.Println("Response: here ->", success)

			}

			if fileType == "image" {
				// Get the file path of the image message

				imageBytes, err := os.ReadFile(newFilename)
				if err != nil {
					fmt.Println(err)
					return
				}

				resps, err := mycli.WAClient.Upload(context.Background(), imageBytes, whatsmeow.MediaImage)
				if err != nil {
					fmt.Println(err)
					return
				}

				// Create a message object with the image data.
				imageMsg := &waProto.ImageMessage{
					Caption:       proto.String(captionTxt),
					Mimetype:      proto.String("image/png"),
					URL:           &resps.URL,
					DirectPath:    &resps.DirectPath,
					MediaKey:      resps.MediaKey,
					FileEncSHA256: resps.FileEncSHA256,
					FileSHA256:    resps.FileSHA256,
					FileLength:    &resps.FileLength,
				}
				_, err = mycli.WAClient.SendMessage(context.Background(), userJid, &waProto.Message{
					ImageMessage: imageMsg,
				})
				if err != nil {
					fmt.Println(err)
					return
				}
			}

			if fileType == "video" {

				imageBytes, err := os.ReadFile(newFilename)
				if err != nil {
					fmt.Println(err)
					return
				}

				resps, err := mycli.WAClient.Upload(context.Background(), imageBytes, whatsmeow.MediaVideo)
				if err != nil {
					fmt.Println(err)
					return
				}

				videoMsg := &waProto.VideoMessage{
					Caption:       proto.String(captionTxt),
					Mimetype:      proto.String("video/mp4"),
					URL:           proto.String(resps.URL),
					DirectPath:    proto.String(resps.DirectPath),
					MediaKey:      resps.MediaKey,
					FileEncSHA256: resps.FileEncSHA256,
					FileSHA256:    resps.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(newFilename))),
				}
				_, err = mycli.WAClient.SendMessage(context.Background(), userJid, &waProto.Message{
					VideoMessage: videoMsg,
				})
				if err != nil {
					fmt.Println(err)
					return
				}

			}

			if fileType == "doc" {

				imageBytes, err := os.ReadFile(newFilename)
				if err != nil {
					fmt.Println(err)
					return
				}

				resps, err := mycli.WAClient.Upload(context.Background(), imageBytes, whatsmeow.MediaDocument)
				if err != nil {
					fmt.Println(err)
					return
				}
				videoMsg := &waProto.DocumentMessage{
					Caption:       proto.String(captionTxt),
					Mimetype:      proto.String(contentType),
					URL:           proto.String(resps.URL),
					DirectPath:    proto.String(resps.DirectPath),
					MediaKey:      resps.MediaKey,
					FileEncSHA256: resps.FileEncSHA256,
					FileSHA256:    resps.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(newFilename))),
				}
				_, err = mycli.WAClient.SendMessage(context.Background(), userJid, &waProto.Message{
					DocumentMessage: videoMsg,
				})
				if err != nil {
					fmt.Println(err)
					return
				}

			}
			fmt.Fprintf(w, "Success")

		} else if err == http.ErrMissingFile {
			message := r.FormValue("message")
			if message == "" {
				http.Error(w, "message is empty", http.StatusUnprocessableEntity)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// w.Write(responseJSON)
			// Send the response.

			// Send a message back to the user using WhatsApp client
			// msg := message
			responsez := &waProto.Message{Conversation: proto.String(string(message))}
			_, errs := mycli.WAClient.SendMessage(context.Background(), userJid, responsez)

			if errs != nil {
				fmt.Printf("Failed to send message to user %s: %s\n", userJid.User, errs)
			} else {
				fmt.Printf("Message sent successfully to user %s\n", userJid.User)
			}
			fmt.Fprintf(w, "Success")
			return

		}

	} else {
		// Respond with a method not allowed error for other request methods
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	// WhatsApp client setup
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	mycli := &MyClient{WAClient: client}
	mycli.register()

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				qrCode := evt.Code
				// Print the QR code string to the standard output
				fmt.Println("qr code:>>", qrCode)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// HTTP server setup
	http.HandleFunc("/server", mycli.handler)

	port := "8080"
	address := fmt.Sprintf(":%s", port)

	fmt.Printf("Server listening on...\n", address)
	err = http.ListenAndServe(address, nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
