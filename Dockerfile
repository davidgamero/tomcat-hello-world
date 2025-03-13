FROM tomcat:9.0

# Remove the default Tomcat web applications
RUN rm -rf /usr/local/tomcat/webapps/*

# (Optional) Ensure the ROOT webapp directory exists
RUN mkdir -p /usr/local/tomcat/webapps/ROOT

# Copy the web application files into the ROOT context
COPY web/ /usr/local/tomcat/webapps/ROOT/

# Expose the default Tomcat port
EXPOSE 8080

# Start Tomcat in run mode
CMD ["catalina.sh", "run"]