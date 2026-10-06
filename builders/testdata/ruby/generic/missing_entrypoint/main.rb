# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

require 'rubygems'
require 'bundler/setup'
require "sinatra"

configure do
  set :port, ENV['PORT']
  set :bind, '0.0.0.0'
end

get "/" do
  "PASS"
end

get '/env' do
  want = params['want']
  return "FAIL: ?want must not be empty" unless want

  got = ENV['FOO']
  return "FAIL: $FOO must not be empty" unless got
  return "FAIL: $FOO=#{got}, want #{want}" unless got.start_with?(want)

  "PASS"
end

get '/version' do
  want = params['want']
  return "FAIL: ?want must not be empty" unless want

  if RUBY_VERSION != want
    return "FAIL: RUBY_VERSION=#{RUBY_VERSION}, want #{want}"
  end

  "PASS"
end
