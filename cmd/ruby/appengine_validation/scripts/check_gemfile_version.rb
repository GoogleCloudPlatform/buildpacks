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

# Check preconditions related to gemfile content.
#
# Required argument: path to the gemfile.
#
# A zero exit code indicates preconditions are satisfied, and building for GAE
# can continue. A nonzero exit code indicates preconditions are not satisfied,
# and building should terminate with an error.
# In either case, the script may output a message to stdout (an error message
# for the error case, or an info/warning message for the success case.)

require "bundler"

def check_gemfile gemfile
  runtime_version = /^\d+\.\d+/.match(RUBY_VERSION)[0]
  gemdef = Bundler::Definition.build gemfile, nil, nil
  ruby_dependency = gemdef.ruby_version
  return unless ruby_dependency
  ruby_dependency_str = ruby_dependency.versions.join(', ').inspect
  if ruby_dependency.diff(Bundler::RubyVersion.new(RUBY_VERSION, nil, nil, nil))
    puts "Your #{gemfile} cannot restrict the Ruby version to #{ruby_dependency_str}" \
          " because this runtime uses Ruby #{RUBY_VERSION}." \
          " Consider instead:  ruby '~> #{runtime_version}.0'"
    exit 1
  end
  if ruby_dependency.diff(Bundler::RubyVersion.new("#{runtime_version}.99", nil, nil, nil))
    puts "Your #{gemfile} cannot restrict the Ruby version to #{ruby_dependency_str}" \
          " because App Engine may upgrade the Ruby patchlevel from #{RUBY_VERSION}." \
          " Consider instead:  ruby '~> #{runtime_version}.0'"
    exit 1
  end
end

check_gemfile(*ARGV)
