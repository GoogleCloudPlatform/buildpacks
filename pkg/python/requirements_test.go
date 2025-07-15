// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package python

import (
	"testing"
)

func TestContainsGunicorn(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "gunicorn_present",
			str:  "gunicorn==19.9.0\nflask\n",
			want: true,
		},
		{
			name: "gunicorn_present_with_comment",
			str:  "gunicorn #my-comment\nflask\n",
			want: true,
		},
		{
			name: "gunicorn_present_second_line",
			str:  "flask\ngunicorn==19.9.0",
			want: true,
		},
		{
			name: "no_gunicorn_present",
			str:  "gunicorn-logging==0.1.0\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_egg_present",
			str:  "git+git://github.com/gunicorn@master#egg=gunicorn\nflask\n",
			want: true,
		},
		{
			name: "gunicorn_egg_not_present",
			str:  "git+git://github.com/gunicorn-logging@master#egg=gunicorn-logging\nflask\n",
			want: false,
		},
		{
			name: "uvicorn_present",
			str:  "uvicorn==3.9.0\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_present_with_comment",
			str:  "uvicorn #my-comment\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_present_with_standard_version",
			str:  "uvicorn[standard] #my-comment\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_present_second_line",
			str:  "fastapi\nuvicorn==3.9.0",
			want: false,
		},
		{
			name: "no_uvicorn_present",
			str:  "uvicorn-logging==0.1.0\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_egg_present",
			str:  "git+git://github.com/uvicorn@master#egg=uvicorn\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_egg_not_present",
			str:  "git+git://github.com/uvicorn-logging@master#egg=uvicorn-logging\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_and_gunicorn_present",
			str:  "uvicorn==3.9.0\ngunicorn==19.9.0\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_and_gunicorn_egg_present",
			str:  "git+git://github.com/uvicorn@master#egg=uvicorn\ngit+git://github.com/gunicorn@master#egg=gunicorn\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_and_gunicorn_egg_not_present",
			str:  "git+git://github.com/uvicorn-logging@master#egg=uvicorn-logging\ngit+git://github.com/gunicorn-logging@master#egg=gunicorn-logging\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_and_gunicorn_present_second_line",
			str:  "fastapi\nuvicorn==3.9.0\ngunicorn==19.9.0",
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPackage(tc.str, "gunicorn")
			if got != tc.want {
				t.Errorf("containsPackage(gunicorn) got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestContainsUvicorn(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "gunicorn_present",
			str:  "gunicorn==19.9.0\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_present_with_comment",
			str:  "gunicorn #my-comment\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_present_second_line",
			str:  "flask\ngunicorn==19.9.0",
			want: false,
		},
		{
			name: "no_gunicorn_present",
			str:  "gunicorn-logging==0.1.0\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_egg_present",
			str:  "git+git://github.com/gunicorn@master#egg=gunicorn\nflask\n",
			want: false,
		},
		{
			name: "gunicorn_egg_not_present",
			str:  "git+git://github.com/gunicorn-logging@master#egg=gunicorn-logging\nflask\n",
			want: false,
		},
		{
			name: "uvicorn_present",
			str:  "uvicorn==3.9.0\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_present_with_comment",
			str:  "uvicorn #my-comment\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_present_with_standard_version",
			str:  "uvicorn[standard] #my-comment\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_present_second_line",
			str:  "fastapi\nuvicorn==3.9.0",
			want: true,
		},
		{
			name: "no_uvicorn_present",
			str:  "uvicorn-logging==0.1.0\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_egg_present",
			str:  "git+git://github.com/uvicorn@master#egg=uvicorn\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_egg_not_present",
			str:  "git+git://github.com/uvicorn-logging@master#egg=uvicorn-logging\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_and_gunicorn_present",
			str:  "uvicorn==3.9.0\ngunicorn==19.9.0\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_and_gunicorn_egg_present",
			str:  "git+git://github.com/uvicorn@master#egg=uvicorn\ngit+git://github.com/gunicorn@master#egg=gunicorn\nfastapi\n",
			want: true,
		},
		{
			name: "uvicorn_and_gunicorn_egg_not_present",
			str:  "git+git://github.com/uvicorn-logging@master#egg=uvicorn-logging\ngit+git://github.com/gunicorn-logging@master#egg=gunicorn-logging\nfastapi\n",
			want: false,
		},
		{
			name: "uvicorn_and_gunicorn_present_second_line",
			str:  "fastapi\nuvicorn==3.9.0\ngunicorn==19.9.0",
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPackage(tc.str, "uvicorn")
			if got != tc.want {
				t.Errorf("containsPackage(uvicorn) got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestContainsGradio(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "gradio_present",
			str:  "gradio==19.9.0\nfastapi\n",
			want: true,
		},
		{
			name: "gradio_present_with_comment",
			str:  "gradio #my-comment\nfastapi\n",
			want: true,
		},
		{
			name: "gradio_present_second_line",
			str:  "fastapi\ngradio==19.9.0",
			want: true,
		},
		{
			name: "no_gradio_present",
			str:  "gradio-logging==0.1.0\nfastapi\n",
			want: false,
		},
		{
			name: "gradio_egg_present",
			str:  "git+git://github.com/gradio@master#egg=gradio\nfastapi\n",
			want: true,
		},
		{
			name: "gradio_egg_not_present",
			str:  "git+git://github.com/gradio-logging@master#egg=gradio-logging\nfastapi\n",
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPackage(tc.str, "gradio")
			if got != tc.want {
				t.Errorf("containsPackage(gradio) got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestContainsStreamlit(t *testing.T) {
	testCases := []struct {
		name string
		str  string
		want bool
	}{
		{
			name: "streamlit_present",
			str:  "streamlit==19.9.0\nfastapi\n",
			want: true,
		},
		{
			name: "streamlit_present_with_comment",
			str:  "streamlit #my-comment\nfastapi\n",
			want: true,
		},
		{
			name: "streamlit_present_second_line",
			str:  "fastapi\nstreamlit==19.9.0",
			want: true,
		},
		{
			name: "no_streamlit_present",
			str:  "streamlit-logging==0.1.0\nfastapi\n",
			want: false,
		},
		{
			name: "streamlit_egg_present",
			str:  "git+git://github.com/streamlit@master#egg=streamlit\nfastapi\n",
			want: true,
		},
		{
			name: "streamlit_egg_not_present",
			str:  "git+git://github.com/streamlit-logging@master#egg=streamlit-logging\nfastapi\n",
			want: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPackage(tc.str, "streamlit")
			if got != tc.want {
				t.Errorf("containsPackage(streamlit) got %t, want %t", got, tc.want)
			}
		})
	}
}
