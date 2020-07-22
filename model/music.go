package model

type Music struct {
    Title string `json:"title"`
    LinkUrl string `json:"linkUrl"`
    Quality string `json:"quality"`
    Singer string `json:"singer"`
    Album string `json:"album"`
}