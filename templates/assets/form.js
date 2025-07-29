;(function() {
  document.addEventListener('DOMContentLoaded', function() {
    const mainElm = document.querySelector('main.form')
    if(!mainElm){
      return;
    }

    function resize(){
      const bg = mainElm.querySelector('.bg');
      if(bg){
        bg.innerHTML = '<span></span>'.repeat((window.innerWidth / 100) * (window.innerHeight / 100) * (window.innerWidth < 800 ? 10 : 6.25));
      }
    }
    resize();
    window.addEventListener('resize', resize, {passive: true});

    return

    let sessionToken = null;
    let loginUserID = null;

    async function sendFormData(action, data){
      if(typeof data !== 'object'){data = {};}
      data.action = action;
      data.session = sessionToken;

      const res = await fetch('/api/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
        body: JSON.stringify(data),
      })

      if(!res.ok){
        setStatus(`Error: ${res.status}! Please Try Again.`);
        return null;
      }

      const json = await res.json();

      if(json?.session){
        sessionToken = json.session;
        delete json.session;
      }

      if(!json || !json.success){
        setStatus(json?.msg || 'An Error Occurred! Please Try Again.');
        return null;
      }

      if(json.msg && json.msg !== ''){
        setStatus(json.msg);
      }else{
        setStatus('');
      }

      return json;
    }
    
    const loginForm = document.querySelector('form[action="/login"]');
    if(loginForm){

      // name: method on submit
      // _name: html form string
      // __name: method on append (to add event listeners)
      const loginMethods = {

        _login: `
          ${formInput('tel', 'phone', 'Phone Number', 'required')}
          ${formSubmit('Send OTP')}
        `,
        login: async function(values){
          let phone = values['phone'];

          if(!phone || (!phone.match(/^(\+[0-9]{1,3}\s?|)(\(?[0-9]{3}\)?[-\s]?)[0-9]{3}-?[0-9]{4}$/) && !phone.match(/^[^\s@]+@[^\s@]+\.[^\s@]+$/))){
            setStatus('Invalid Phone Number!');
            return;
          }

          const json = await sendFormData('send-otp', {phone: phone});
          if(!json){
            return;
          }

          if(json.email){
            const input = loginForm.querySelector('input[name="phone"]');
            if(input){
              input.value = json.email;
            }
          }else if(json.budgetSMS){
            appendForm('budgetlogin');
            return;
          }

          appendForm('otp');
        },

        _otp: `
          ${formInput('text', 'otp', 'OTP', 'required maxlength="6"')}
          ${formSubmit('Verify OTP')}
        `,
        __otp: function(){
          // auto submit when filled
          const otpInput = loginForm.querySelector('input[name="otp"]');
          if(otpInput){
            otpInput.addEventListener('input', function(){
              if(otpInput.value.length === 6){
                submitLoginForm();
              }
            }, {passive: true});
          }
        },
        otp: async function(values){
          let otp = values['otp'];
          if(!otp || !otp.match(/^[0-9]{6}$/)){
            setStatus('Invalid OTP!');
            return;
          }

          let phone = values['phone'];
          if(!phone || (!phone.match(/^(\+[0-9]{1,3}\s?|)(\(?[0-9]{3}\)?[-\s]?)[0-9]{3}-?[0-9]{4}$/) && !phone.match(/^[^\s@]+@[^\s@]+\.[^\s@]+$/))){
            setStatus('Invalid Phone Number!');
            return;
          }

          const json = await sendFormData('verify-otp', {phone: phone, otp: otp});
          if(!json){
            return;
          }

          // if current user, ask for a password
          if(!json.signup){
            loadForm('passwd');
            return;
          }

          // if new user, ask for signup info
          loadForm('signup');
        },

        _passwd: `
          ${formInput('password', 'password', 'Password', 'required maxlength="50"')}
          ${formSubmit('Login')}
        `,
        __passwd: function(){
          //todo: may update header in OTP step if user number recognized
          setHeading('Login');
        },
        passwd: function(values){
          //todo: verify user password
        },

        _signup: `
          ${formInput('text', 'username', 'Username', 'required maxlength="50"')}
          ${formInput('text', 'email', 'Email', 'required maxlength="50"')}

          ${formInput('password', 'password', 'Password', 'required maxlength="50"')}
          ${formInput('password', 'confirm-password', 'Confirm Password', 'required maxlength="50"')}
          ${formSubmit('Create Account')}
        `,
        __signup: function(){
          //todo: may update header in OTP step if user number not recognized
          setHeading('Signup');
        },
        signup: function(values){
          let passwd = values['password'];
          if(!passwd){
            setStatus('A Password Is Required!');
            return;
          }

          { // verify secure password (additional verification may be done on the server)
            if(passwd.length < 8){
              setStatus('Password Must Be At Least 8 Characters!');
              return;
            }
  
            if(!passwd.match(/[A-Z]/)){
              setStatus('Password Must Contain A Capital Letter!');
              return;
            }
  
            if(!passwd.match(/[a-z]/)){
              setStatus('Password Must Contain A Lowercase Letter!');
              return;
            }
  
            if(!passwd.match(/[0-9]/)){
              setStatus('Password Must Contain A Number!');
              return;
            }
  
            if(!passwd.match(/[^\w]/)){
              setStatus('Password Must Contain A Special Character!');
              return;
            }
  
            if(!values['confirm-password'] || passwd !== values['confirm-password']){
              setStatus('Passwords Do Not Match!');
              return;
            }
          }

          //todo: verify and send data to server
          // still need username and email (may allow verifying email later)
          // or may require verified email for potential account recovery if needed

          console.log(passwd, values)

          //todo: continue form (make additional info optional)
          loadForm('userinfo');
        },

        _userinfo: `
          ${formInput('text', 'name', 'Name', 'maxlength="50"')}

          <div class="wrapper">
            <label for="fm-icon">Upload Icon</label>
            <input type="file" id="fm-icon" name="icon"/>
          </div>

          <fieldset>
            <legend>Gender</legend>

            <div class="wrapper">
              <input type="radio" id="fm-gender-male" name="gender" value="m"/>
              <label for="fm-gender-male">Male</label>
            </div>

            <div class="wrapper">
              <input type="radio" id="fm-gender-female" name="gender" value="f"/>
              <label for="fm-gender-female">Female</label>
            </div>
          </fieldset>

          ${formSubmit('Update')}

          <!-- todo: add optional skip button for this step -->
        `,
        __userinfo: function(){
          setHeading('Finish Setup');
        },
        userinfo: function(values){
          //todo: send data to server

          console.log(values)

        },

        _budgetlogin: `
          <span class="info">We are on a budget! Please verify your Email, before verifying SMS.</span>
          ${formInput('text', 'email', 'Email', 'required maxlength="50"')}
          ${formSubmit('Email OTP')}
        `,
        budgetlogin: async function(values){
          let phone = values['phone'];
          let email = values['email'];

          if(!phone || !phone.match(/^(\+[0-9]{1,3}\s?|)(\(?[0-9]{3}\)?[-\s]?)[0-9]{3}-?[0-9]{4}$/)){
            setStatus('Invalid Phone Number!');
            return;
          }

          if(!email || !email.match(/^[^\s@]+@[^\s@]+\.[^\s@]+$/)){
            setStatus('Invalid Email Address!');
            return;
          }

          const json = await sendFormData('send-budget-otp', {phone: phone, email: email});
          if(!json){
            return;
          }

          loginForm.querySelector('span.info')?.remove();

          appendForm('budgetotp');
        },

        _budgetotp: `
          ${formInput('text', 'otp', 'OTP', 'required maxlength="6"')}
          ${formSubmit('Verify OTP')}
        `,
        __budgetotp: function(){
          // auto submit when filled
          const otpInput = loginForm.querySelector('input[name="otp"]');
          if(otpInput){
            otpInput.addEventListener('input', function(){
              if(otpInput.value.length === 6){
                submitLoginForm();
              }
            }, {passive: true});
          }
        },
        budgetotp: async function(values){
          let otp = values['otp'];
          let phone = values['phone'];
          let email = values['email'];

          if(!otp || !otp.match(/^[0-9]{6}$/)){
            setStatus('Invalid OTP!');
            return;
          }

          if(!phone || !phone.match(/^(\+[0-9]{1,3}\s?|)(\(?[0-9]{3}\)?[-\s]?)[0-9]{3}-?[0-9]{4}$/)){
            setStatus('Invalid Phone Number!');
            return;
          }

          if(!email || !email.match(/^[^\s@]+@[^\s@]+\.[^\s@]+$/)){
            setStatus('Invalid Email Address!');
            return;
          }

          const json = await sendFormData('verify-budget-otp', {phone: phone, email: email, otp: otp});
          if(!json){
            return;
          }

          if(json.budgetSignup){
            const inputOTP = loginForm.querySelector('input[name="otp"]');
            if(inputOTP){
              inputOTP.setAttribute('readonly', '');
              inputOTP.value = json["otp"];

              appendForm('budgetotptext');

              const adminNumber = loginForm.querySelector('a.admin-number');
              if(adminNumber){
                adminNumber.textContent = json.budgetSignup;
                adminNumber.href = `tel:${json.budgetSignup.replace(/[^0-9+]/g, '')}`;
              }

              return;
            }
          }

          loadForm('signup');

          loginForm.querySelector('input[name="email"]')?.parentNode.remove();
        },

        _budgetotptext: `
          <span class="info">Text your Email and the OTP to <a class="admin-number"></a> for manual account approval.</span>
          ${formSubmit('Next')}
        `,
        budgetotptext: function(values){
          loadForm('signup');

          loginForm.querySelector('input[name="email"]')?.parentNode.remove();
        },

      };

      function formInput(type, name, label, atts){
        return `
        <div class="wrapper">
          <label for="fm-${name}">${label}</label>
          <input type="${type}" id="fm-${name}" name="${name}" ${atts}/>
        </div>`;
      }

      function formSubmit(value){
        return `<input type="submit" value="${value}"/>`;
      }

      function loadForm(name){
        if(typeof name !== 'string'){return;}
        name = name.replace(/[^\w]/g, '');
  
        if(loginMethods['_'+name]){
          loginForm.innerHTML = loginMethods['_'+name];
          loginForm.setAttribute('mode', name);
        }
  
        if(typeof loginMethods['__'+name] === 'function'){
          loginMethods['__'+name]();
        }
      }

      function appendForm(name){
        if(typeof name !== 'string'){return;}
        name = name.replace(/[^\w]/g, '');
  
        if(loginMethods['_'+name]){
          loginForm.querySelector('input[type="submit"]')?.remove();
          loginForm.querySelectorAll('input').forEach(function(elm){
            elm.setAttribute('readonly', '');
          });
  
          const wrapper = document.createElement('div');
          wrapper.classList.add('js-wrapper');
          wrapper.innerHTML = loginMethods['_'+name];
          loginForm.appendChild(wrapper);
          loginForm.setAttribute('mode', name);
        }
  
        if(typeof loginMethods['__'+name] === 'function'){
          loginMethods['__'+name]();
        }
      }

      const statusMsg = document.querySelector('main .login .status');
      function setStatus(text){
        if(statusMsg){
          statusMsg.textContent = text || '';
        }
      }

      const loginHeading = document.querySelector('main .login h1');
      function setHeading(text){
        if(loginHeading){
          loginHeading.textContent = text || 'Login / Signup';
        }
      }

      //todo: check for hidden session key, an store it in js
      // also remember to restore hidden session key when loading new form inputs

      function submitLoginForm(e){
        e?.preventDefault();
        setStatus('');
        const mode = loginForm.getAttribute('mode').replace(/[^\w]/g, '');

        const formValues = {};
        loginForm.querySelectorAll('input:not([type="submit"])').forEach(function(elm){
          const name = elm.name.replace(/[^\w_-]/g, '');

          if(elm.type === 'hidden' && name === 'session'){
            sessionToken = elm.value;
            elm.remove();
            return;
          }

          if(elm.type === 'radio'){
            if(elm.checked){
              formValues[name] = elm.value;
            }else if(!formValues[name]){
              formValues[name] = '';
            }
            return;
          }else if(elm.type === 'checkbox'){
            if(elm.checked){
              formValues[name] = true;
            }else{
              formValues[name] = false;
            }
            return;
          }

          formValues[name] = elm.value;
        });

        if(!sessionToken){
          setStatus('Form Session Token Not Found!');
          return;
        }

        if(typeof loginMethods[mode] === 'function'){
          (async function(){
            loadForm(await loginMethods[mode](formValues));
          })();
        }
      }
      loginForm.addEventListener('submit', submitLoginForm);

      setInterval(function(){
        loginForm.querySelectorAll('input:not([form-js-ready])').forEach(function(elm){
          elm.setAttribute('form-js-ready', '');

          elm.addEventListener('input', function(){
            setStatus('');
          }, {passive: true});
        });
      }, 1000);
    }

  });
})();
