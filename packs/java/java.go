package java

import (
  "encoding/xml"
)

type MavenProject struct {
  XMLName xml.Name `xml:"project"`
  ModelVersion string `xml:"modelVersion"`
  GroupId string `xml:"groupId"`
  ArtifactId string `xml:"artifactId"`
  Packaging string `xml:"packaging"`
  Version string `xml:"version"`
}
