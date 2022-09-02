package chartutil

import (
	"archive/tar"
	"compress/gzip"

	"helm.sh/helm/v3/pkg/chart"
)

type SaveIntoTarOptions struct {
	Prefix string
}

func SaveIntoTar(out *tar.Writer, c *chart.Chart, opts SaveIntoTarOptions) error {
	return writeTarContents(out, c, opts.Prefix)
}

func SetGzipWriterMeta(zipper *gzip.Writer) {
	zipper.Header.Extra = headerBytes
	zipper.Header.Comment = "Helm"
}
