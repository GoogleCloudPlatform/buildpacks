using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;

namespace cs_web
{
    public class Startup
    {
        // This method gets called by the runtime. Use this method to add services to the container.
        // For more information on how to configure your application, visit https://go.microsoft.com/fwlink/?LinkID=398940
        public void ConfigureServices(IServiceCollection services)
        {
        }

        // This method gets called by the runtime. Use this method to configure the HTTP request pipeline.
        public void Configure(IApplicationBuilder app, IWebHostEnvironment env)
        {

            app.UseRouting();

            app.UseEndpoints(endpoints =>
            {
                endpoints.MapGet("/", async context =>
                {
                    await context.Response.WriteAsync("PASS");
                });

                endpoints.MapGet("/version", async context =>
                {
                  string got = System.Environment.Version.ToString();
                  string want = context.Request.Query["want"];

                  if (want == null)
                  {
                      await context.Response.WriteAsync("FAIL: ?want must be set to a version");
                  }
                  else if (got != want)
                  {
                      await context.Response.WriteAsync($"FAIL: current version: {got}; want {want}");
                  }
                  else
                  {
                      await context.Response.WriteAsync("PASS");
                  }
                });
            });
        }
    }
}
