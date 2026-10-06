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

using Newtonsoft.Json;

namespace Dep
{

  public static class Common
  {

    public class Item
    {
      public int Id { get; set; }
    }

    public static string Message()
    {
      Item item = new Item
      {
        Id = 1,
      };
      // This value is intentionally ignored. It is only used to verify imports work.
      string json = JsonConvert.SerializeObject(item, Formatting.Indented);
      return "PASS";
    }
  }
}
