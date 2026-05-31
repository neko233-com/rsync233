package rsync233

import "testing"

func TestParseEndpointLocalWindowsDrive(t *testing.T) {
	ep, err := ParseEndpoint(`C:\tmp\src`)
	if err != nil {
		t.Fatal(err)
	}
	if ep.IsRemote() {
		t.Fatalf("windows drive path parsed as remote: %+v", ep)
	}
}

func TestParseEndpointSSHURL(t *testing.T) {
	ep, err := ParseEndpoint("ssh://deploy@example.com:2222/var/www/")
	if err != nil {
		t.Fatal(err)
	}
	if !ep.IsRemote() || ep.User != "deploy" || ep.Host != "example.com" || ep.Port != 2222 || !ep.Trailing {
		t.Fatalf("unexpected endpoint: %+v", ep)
	}
}

func TestParseEndpointScpLike(t *testing.T) {
	ep, err := ParseEndpoint("deploy@example.com:/var/www")
	if err != nil {
		t.Fatal(err)
	}
	if !ep.IsRemote() || ep.User != "deploy" || ep.Host != "example.com" || ep.Path != "/var/www" {
		t.Fatalf("unexpected endpoint: %+v", ep)
	}
}
