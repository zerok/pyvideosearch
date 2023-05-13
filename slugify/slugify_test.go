package slugify_test

import (
	"testing"

	"github.com/zerok/pyvideosearch/slugify"
)

func TestSlugify(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{
			input:    "Carl Meyer about Django @ Instagram at Django: Under The Hood 2016",
			expected: "carl-meyer-about-django-instagram-at-django-under-the-hood-2016",
		},
		{
			input:    "Unicode is \u2202\u00a9 in python 2, but better in python 3",
			expected: "unicode-is-c-in-python-2-but-better-in-python-3",
		},
		{
			input:    "Micropython \u0434\u043b\u044f \u043a\u0432\u0435\u0441\u0442\u043e\u0432 \u0432 \u0440\u0435\u0430\u043b\u044c\u043d\u043e\u0441\u0442\u0438 \u0438 \u0430\u0440\u043a\u0430\u0434\u043d\u044b\u0445 \u0438\u0433\u0440 / \u041d\u0438\u043a\u0438\u0442\u0430 \u041b\u0435\u0432\u043e\u043d\u043e\u0432\u0438\u0447 (\u041a\u0412\u0415\u0421\u0422\u041e\u0414\u0415\u041b\u042b)",
			expected: "micropython-dlia-kvestov-v-realnosti-i-arkadnykh-igr-nikita-levonovich-kvestodely",
		},
		{
			input:    "\u0410 \u0447\u0442\u043e, \u0435\u0441\u043b\u0438 \u0431\u0435\u0437 Python? Julia \u0434\u043b\u044f \u043c\u0430\u0448\u0438\u043d\u043d\u043e\u0433\u043e \u043e\u0431\u0443\u0447\u0435\u043d\u0438\u044f \u0438 \u0432\u043e\u043e\u0431\u0449\u0435 / \u0413\u043b\u0435\u0431 \u0418\u0432\u0430\u0448\u043a\u0435\u0432\u0438\u0447",
			expected: "a-chto-esli-bez-python-julia-dlia-mashinnogo-obucheniia-i-voobshche-gleb-ivashkevich",
		},
		{
			input:    "Anaconda\u74b0\u5883\u904b\u7528TIPS \u301cAnaconda\u306e\u74b0\u5883\u69cb\u7bc9\u306b\u3064\u3044\u3066\u77e5\u308b\u30fb\u8cea\u554f\u306b\u7b54\u3048\u3089\u308c\u308b\u3088\u3046\u306b\u306a\u308b\u301c",
			expected: "anacondahuan-jing-yun-yong-tips-anacondanohuan-jing-gou-zhu-nitsuitezhi-ruzhi-wen-nida-erareruyouninaru",
		},
	}

	for _, test := range testcases {
		output := slugify.Slugify(test.input)
		if output != test.expected {
			t.Fatalf("\nExpected: `%s`\nActual:   `%s`", test.expected, output)
		}
	}
}
