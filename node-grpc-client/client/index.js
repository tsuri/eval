'use strict';

module.exports = {
  getEmployeeDetails(client, employeeId) {
    client.Eval({number: employeeId}, function (err, response) {
      if (err) {
        console.log(err)
      }
      console.log('Employee Details for Employee Id:', employeeId, '\n',
          response);
    });
  }
}
