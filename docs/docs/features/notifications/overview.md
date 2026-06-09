---
title: Notifications
description: Display native system notifications with action buttons and text input
image: ../../../assets/images/og-image.png
---

[Skip to content](#%5Ftop) 

# Notifications

## Introduction

[Section titled “Introduction”](#introduction)

Wails provides a comprehensive cross-platform notification system for desktop applications. This service allows you to display native system notifications, with support for interactive elements like action buttons and text input fields.

## Basic Usage

[Section titled “Basic Usage”](#basic-usage)

### Creating the Service

[Section titled “Creating the Service”](#creating-the-service)

First, initialize the notifications service:

```

import "github.com/wailsapp/wails/v3/pkg/application"

import "github.com/wailsapp/wails/v3/pkg/services/notifications"


// Create a new notification service

notifier := notifications.New()


//Register the service with the application

app := application.New(application.Options{

    Services: []application.Service{

        application.NewService(notifier),

    },

})


```

## Notification Authorization

[Section titled “Notification Authorization”](#notification-authorization)

Notifications on macOS require user authorization. Request and check authorization:

```

authorized, err := notifier.CheckNotificationAuthorization()

if err != nil {

    // Handle authorization error

}

if authorized {

    // Send notifications

} else {

    // Request authorization

    authorized, err = notifier.RequestNotificationAuthorization()

}


```

On Windows and Linux this always returns `true`.

## Notification Types

[Section titled “Notification Types”](#notification-types)

### Basic Notifications

[Section titled “Basic Notifications”](#basic-notifications)

Send a basic notification with a unique id, title, optional subtitle (macOS and Linux), and body text to users:

```

notifier.SendNotification(notifications.NotificationOptions{

    ID: "unique-id",

    Title: "New Calendar Invite",

    Subtitle: "From: Jane Doe", // Optional

    Body: "Tap to view the event",

})


```

### Interactive Notifications

[Section titled “Interactive Notifications”](#interactive-notifications)

Send a notification with action buttons and text inputs. These notifications require a notification category to be resgistered first:

```

// Define a unique category id

categoryID := "unique-category-id"


// Define a category with actions

category := notifications.NotificationCategory{

    ID: categoryID,

    Actions: []notifications.NotificationAction{

        {

            ID:    "OPEN",

            Title: "Open",

        },

        {

            ID:          "ARCHIVE",

            Title:       "Archive",

            Destructive: true,  /* macOS-specific */

        },

    },

    HasReplyField:    true,

    ReplyPlaceholder: "message...",

    ReplyButtonTitle: "Reply",

}


// Register the category

notifier.RegisterNotificationCategory(category)


// Send an interactive notification with the actions registered in the provided category

notifier.SendNotificationWithActions(notifications.NotificationOptions{

    ID:         "unique-id",

    Title:      "New Message",

    Subtitle:   "From: Jane Doe",

    Body:       "Are you able to make it?",

    CategoryID: categoryID,

})


```

## Notification Responses

[Section titled “Notification Responses”](#notification-responses)

Process user interactions with notifications:

```

notifier.OnNotificationResponse(func(result notifications.NotificationResult) {

    response := result.Response

    fmt.Printf("Notification %s was actioned with: %s\n", response.ID, response.ActionIdentifier)


    if response.ActionIdentifier == "TEXT_REPLY" {

        fmt.Printf("User replied: %s\n", response.UserText)

    }


    if data, ok := response.UserInfo["sender"].(string); ok {

        fmt.Printf("Original sender: %s\n", data)

    }


    // Emit an event to the frontend

    app.Event.Emit("notification", result.Response)

})


```

## Notification Customisation

[Section titled “Notification Customisation”](#notification-customisation)

### Custom Metadata

[Section titled “Custom Metadata”](#custom-metadata)

Basic and interactive notifications can include custom data:

```

notifier.SendNotification(notifications.NotificationOptions{

    ID: "unique-id",

    Title: "New Calendar Invite",

    Subtitle: "From: Jane Doe", // Optional

    Body: "Tap to view the event",

    Data: map[string]interface{}{

        "sender": "jane.doe@example.com",

        "timestamp": "2025-03-10T15:30:00Z",

    }

})


```

## Platform Considerations

[Section titled “Platform Considerations”](#platform-considerations)

* [  macOS ](#tab-panel-89)
* [  Windows ](#tab-panel-90)
* [  Linux ](#tab-panel-91)

On macOS, notifications:

* Require user authorization
* Require the app to be notorized for distribution
* Use system-standard notification appearances
* Support `subtitle`
* Support user text input
* Support the `Destructive` action option
* Automatically handle dark/light mode

On Windows, notifications:

* Use Windows system toast styles
* Adapt to Windows theme settings
* Support user text input
* Support high DPI displays
* Do not support `subtitle`

On Linux, dialog behaviour depends on the desktop environment:

* Use native notifications when available
* Follow desktop environment theme
* Position according to desktop environment rules
* Support `subtitle`
* Do not support user text input

## Best Practices

[Section titled “Best Practices”](#best-practices)

1. Check and request for authorization:  
   * macOS requires user authorization
2. Provide clear and concise notifications:  
   * Use descriptive titles, subtitles, text, and action titles
3. Handle dialog responses appropriately:  
   * Check for errors in notification responses  
   * Provide feedback for user actions
4. Consider platform conventions:  
   * Follow platform-specific notification patterns  
   * Respect system settings

## Examples

[Section titled “Examples”](#examples)

Explore this example:

* [Notifications](https://v3.wails.io/examples/notifications)

## API Reference

[Section titled “API Reference”](#api-reference)

### Service Management

[Section titled “Service Management”](#service-management)

| Method | Description                         |
| ------ | ----------------------------------- |
| New()  | Creates a new notifications service |

### Notification Authorization

[Section titled “Notification Authorization”](#notification-authorization-1)

| Method                             | Description                                              |
| ---------------------------------- | -------------------------------------------------------- |
| RequestNotificationAuthorization() | Requests permission to display notifications (macOS)     |
| CheckNotificationAuthorization()   | Checks current notification authorization status (macOS) |

### Sending Notifications

[Section titled “Sending Notifications”](#sending-notifications)

| Method                                                   | Description                                    |
| -------------------------------------------------------- | ---------------------------------------------- |
| SendNotification(options NotificationOptions)            | Sends a basic notification                     |
| SendNotificationWithActions(options NotificationOptions) | Sends an interactive notification with actions |

### Notification Categories

[Section titled “Notification Categories”](#notification-categories)

| Method                                                      | Description                                |
| ----------------------------------------------------------- | ------------------------------------------ |
| RegisterNotificationCategory(category NotificationCategory) | Registers a reusable notification category |
| RemoveNotificationCategory(categoryID string)               | Removes a previously registered category   |

### Managing Notifications

[Section titled “Managing Notifications”](#managing-notifications)

| Method                                         | Description                                                      |
| ---------------------------------------------- | ---------------------------------------------------------------- |
| RemoveAllPendingNotifications()                | Removes all pending notifications (macOS and Linux only)         |
| RemovePendingNotification(identifier string)   | Removes a specific pending notification (macOS and Linux only)   |
| RemoveAllDeliveredNotifications()              | Removes all delivered notifications (macOS and Linux only)       |
| RemoveDeliveredNotification(identifier string) | Removes a specific delivered notification (macOS and Linux only) |
| RemoveNotification(identifier string)          | Removes a notification (Linux-specific)                          |

### Event Handling

[Section titled “Event Handling”](#event-handling)

| Method                                                           | Description                                     |
| ---------------------------------------------------------------- | ----------------------------------------------- |
| OnNotificationResponse(callback func(result NotificationResult)) | Registers a callback for notification responses |

### Structs and Types

[Section titled “Structs and Types”](#structs-and-types)

#### NotificationOptions

[Section titled “NotificationOptions”](#notificationoptions)

```

type NotificationOptions struct {

    ID         string                 // Unique identifier for the notification

    Title      string                 // Main notification title

    Subtitle   string                 // Subtitle text (macOS and Linux only)

    Body       string                 // Main notification content

    CategoryID string                 // Category identifier for interactive notifications

    Data       map[string]interface{} // Custom data to associate with the notification

}


```

#### NotificationCategory

[Section titled “NotificationCategory”](#notificationcategory)

```

type NotificationCategory struct {

    ID               string                // Unique identifier for the category

    Actions          []NotificationAction  // Button actions for the notification

    HasReplyField    bool                  // Whether to include a text input field

    ReplyPlaceholder string                // Placeholder text for the input field

    ReplyButtonTitle string                // Text for the reply button

}


```

#### NotificationAction

[Section titled “NotificationAction”](#notificationaction)

```

type NotificationAction struct {

    ID          string  // Unique identifier for the action

    Title       string  // Button text

    Destructive bool    // Whether the action is destructive (macOS-specific)

}


```

#### NotificationResponse

[Section titled “NotificationResponse”](#notificationresponse)

```

type NotificationResponse struct {

    ID               string                  // Notification identifier

    ActionIdentifier string                  // Action that was triggered

    CategoryID       string                  // Category of the notification

    Title            string                  // Title of the notification

    Subtitle         string                  // Subtitle of the notification

    Body             string                  // Body text of the notification

    UserText         string                  // Text entered by the user

    UserInfo         map[string]interface{}  // Custom data from the notification

}


```

#### NotificationResult

[Section titled “NotificationResult”](#notificationresult)

```

type NotificationResult struct {

    Response NotificationResponse  // Response data

    Error    error                 // Any error that occurred

}


```

```json
{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Wails","description":"Build beautiful desktop applications using Go and modern web technologies.","url":"https://v3.wails.io","applicationCategory":"DeveloperApplication","downloadUrl":"https://github.com/wailsapp/wails/releases","softwareVersion":"3.0","operatingSystem":["Windows","macOS","Linux"],"offers":{"@type":"Offer","price":"0","priceCurrency":"USD"},"author":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"},"provider":{"@type":"Organization","name":"Wails Contributors","url":"https://github.com/wailsapp/wails"}}
{"@context":"https://schema.org","@type":"Organization","name":"Wails","url":"https://v3.wails.io","logo":"../../../assets/images/favicon.svg","sameAs":["https://github.com/wailsapp/wails","https://x.com/wailsapp","https://discord.gg/JDdSxwjhGf"]}
```
