# Sous quick-start

## What is sous?

Sous is a conventions-based wrapper around the Docker CLI tool that provides some useful defaults and simplifies the build-test-publish cycle. Sous knows how to invoke your application in a local simulated Mesos environment, and can even check your app against contracts required by the platform. This allows you to make sure all the basics are right before you even publish your app, and will hopefully lead to quicker success on the Mesos platform.

_In phase 2, we will be adding support for synchronised global deployments, including remote smoke testing and blue/green deployment pools. This should take a lot of the complexity out of your build and deploy configurations. If you have any ideas in this direction, please [drop me an email](mailto:ssalisbury@opentable.com)_

Sous currently only supports NodeJS projects, but we plan to add support for C#, Java, Ruby, Go, and others very soon.

Once you are happy building your project with Sous, you should remove any Dockerfile left over in your project; Sous does not, and cannot, use your custom Dockerfile, it generates standardised dockerfiles on the fly instead. If you leave your Dockerfile in place, it will simply be ignored by Sous.

### Transparency

Sous is not magic, and doesn't pretend to be, it is designed solely to take the tedium out of Doing It Rite™. As such, it follows a few design principles:

- It logs most of the shell commands it is executing to screen prefixed by `shell>`. This means if someehing goes wrong during a sous operation, you should be able to scroll up and see what it was trying to do, which may be helpful in fixing said issues.
- Taking this further, you can add the `-commands-only` flag to most sous commands to see what shell commands sous would issue for that operation _(coming soon)._
- You can always see up-front what `Dockerfile` sous would build for a given target by issuing `sous dockerfile <target>` inside your project directory. If you omit `target` then `app` is always used as the default.

### Conventions

Sous is not designed to offer endless flexibility in configuration and options. If you need to do things like install complex system-level dependencies inside your container, or customise networking or other OS features, you will have more joy either getting these things into a base image, if they are widely-applicable standards, or rolling your own Dockerfile and build/test/deploy/smoke-test/activate pipeline.

That said, Sous aims to allow as much as possible of what is expressible by conventions in your chosen application stack to contribute to your final built images. For example, NodeJS with NPM and NVM allows you to specify a Node runtime version in your package.json. That is expressed by Sous picking the correct NodeJS base image for that Node version. You can also add a "preinstall" script to your package.json, that too is expressed by adding an implicit `compile` target that the `app` and `test` targets rely on.

### Targets

Sous uses the concept of "targets" to build Docker images with specialised purposes. Here's the current list of targets...

- `app` is the main (default) target, its Docker image will serve your application when it is run with `docker run`
- `test` is a special target whose only job is to run unit tests (`docker run` on this target just runs unit tests and then exits)
- `compile` is an optional special target whose job is to gather all dependencies, and possibly perform other precompile steps. `docker run` on this target will invoke the build process with a special directory mounted at `/artifacts`; the build container's job is to perform all necessary tasks, and then place a complete representation of the application in the `/artifacts` directory. This will then be placed inside the `app` container ready for deployment.
- `smoke` is an optional special target whose job is to run remote smoke tests against individual instances of your deployed application. Smoke tests here can be as simple or as involved as you like, but should at a minimum ensure that your `/health` endpoint is reporting healthy, and that all your other essential routes seem to be working. Eventually this will become a required dependency of the deployment target (coming soon).

### How do I add support for target X?

Sous very deliberately does not require any special files (imagine a "Sousfile") or anything else specific to Sous in your project. One of its key design principles is that a project that works well with Sous will also be a model project in the language/framework in which it was built. To say it another way, the complexity of building projects as Docker images to run on the Mesos Platform is the responsibility of Sous. Your responsibility should be limited to expressing in a way idiomatic to your chosen stack all of your project's dependencies (in the generic sense, including build-time tools and libs, the runtime itself, any run-time modules, etc.). Sous must parse and understand your project in order to generate correct Docker images.

When Sous does not have enough information about an aspect of your project to support a particular target, it will let you know by

a) giving you an error message and exiting when you try to build that target, and
b) listing all targets and their supportedness when you issue a `sous detect`

If you want to add support for a specific target, you can issue a `sous detect <target>` to get specialised help on adding support for that target in your specific project.

Therefore, for each target supported in your stack, there is online help available telling you how to add support for that target. You can issue a `sous detect` to check what targets your project currently supports.

