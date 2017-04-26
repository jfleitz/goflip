/*
JAF 2016-04-27. Lamp Driver controller.
 Uses I2C to control a Gameplan LDU board.
 
 J1 
 1 = Lamp En 1
 2 = Lamp En 2
 3 = Addr 4
 4 = Addr 3
 5 = Lamp En 3
 6 = Key
 7 = Lamp En 4
 8 = Addr 1
 9 = Addr 2
 
 Low PortB (Active High):
 0=lamp Address 0
 1=lamp Address 1
 2=lamp Address 2
 3=lamp Address 3
 
 Low PortC (Active low):
 0=str (Lamp En) 0
 1=str (Lamp En) 1
 2=str (Lamp En) 2
 3=str (Lamp En) 3
 
 LDU logic:
 4 Enables (Active Low) - port C
 4-Bit Data (Active High) - port B
 
 Example for Strobe 0, Lamp 5: 
 PortC = 0x#E
 PortB = 0x#5
 
 Valid Strobe values (0-3)= E (1110) ,D (1101), B (1011), 7 (0111)
 */

#include <avr/io.h>
#include <Wire.h>
#define F_CPU 16000000UL  //16mhz clock
#include <util/delay.h> 
#define LAMP_ADDRESS 0x04 //i2c address here. = 0x04

// Pin 13 has an LED connected on most Arduino boards.
// PB5 is the LED
// give it a name:
int led = 13; 

unsigned char _slowBlink;
unsigned char _fastBlink;
int _timerControl;

//holds the status of the lamp.
//00=off, 01=on, 10=slow blink, 11=fast blink
unsigned char _lightControl[65]; 
unsigned char _brightness[65]; //only used if command 4 is activated for a light
int _lastLamp; //last Lamp ID command sent. Makes debugging easier

//High nibble are the strobes. Active Low.
//Low nibble is the Lamp Data... Active High
unsigned char const _ldu_map[64]= {
  0xE0,0xE1,0xE2,0xE3,0xE4,0xE5,0xE6,0xE7,0xE8,0xE9,
  0xEA,0xEB,0xEC,0xED,0xEE,0xEF,0xD0,0xD1,0xD2,0xD3,0xD4,0xD5,0xD6,0xD7,0xD8,0xD9,
  0xDA,0xDB,0xDC,0xDD,0xDE,0xDF,0xB0,0xB1,0xB2,0xB3,0xB4,0xB5,0xB6,0xB7,0xB8,0xB9,
  0xBA,0xBB,0xBC,0xBD,0xBE,0xBF,0x70,0x71,0x72,0x73,0x74,0x75,0x76,0x77,0x78,0x79,
  0x7A,0x7B,0x7C,0x7D,0x7E,0x7F};


// the setup routine runs once when you press reset:
void setup() {                
  // initialize the digital pin as an output.
  pinMode(led, OUTPUT);     
//  Serial.begin(9600); // start serial for output. Should only have this if debugging
  _lastLamp = 0;

  SetLampAll(0);

  DDRC = DDRC | B00001111; //Port C0-C3 output for Lamp Strobe0-Strobe3
  DDRB = DDRB | B00001111; //Port B0-B3 output for Lamp0-Lamp3

  _timerControl = 0;

  // initialize timer1 
  noInterrupts();           // disable all interrupts
  TCCR1A = 0;
  TCCR1B = 0;
  TCNT1  = 0;

  OCR1A =  15625; //31250;            // compare match register 16MHz/256/2Hz (4 hz)
  TCCR1B |= (1 << WGM12);   // CTC mode
  TCCR1B |= (1 << CS12);    // 256 prescaler 
  TIMSK1 |= (1 << OCIE1A);  // enable timer compare interrupt
  interrupts();             // enable all interrupts

    Wire.begin(LAMP_ADDRESS);                // join i2c bus 
  // register i2 events
  Wire.onReceive(receiveEvent); 
  Wire.onRequest(sendEvent);

 // Serial.println("Initialized!");
}

void SetLampAll(int value){
  int i;

  for(i=0;i<65;i++){
    _lightControl[i]=value;
  }
}

void SetLamp(int lmp, unsigned char data){
  //0=off, 1=on, 2=slow blink, 3=fast blink, 4=brightness
  _lightControl[lmp]=data;
}

// the loop routine runs over and over again forever:
void loop() {

  int i;

  unsigned long int count;
  unsigned char newValue;

  /* Old while(1){
   delay(100);
   }
   */

  /*
   Low PortB:
   0=lamp0
   1=lamp1
   2=lamp2
   3=lamp3
   
   Low PortC:
   0=str0
   1=str1
   2=str2
   3=str3
   */


  while(1){
    //just loop through each lamp value...

    for(i=0;i<65;i++){

      switch(_lightControl[i]){
      case 0:
        if(i<64){
          turnOffAllLamps();
        }
        else{
          turnOffLED();
        }
        break;

      case 1: //on
        if(i<64){
          turnOnLamp(i);
        }
        else{
          turnOnLED();
        }

        break;

      case 2: //slow blink

        if(_slowBlink==0x01){
          if(i<64){
            turnOnLamp(i);
          }
          else{
            turnOnLED();
          }
        }

        else{
          if(i<64){
            turnOffAllLamps();
          }
          else{
            turnOffLED();
          }
        }

        break;

      case 3: //fast blink

        if(_fastBlink==0x01){
          if(i<64){
            turnOnLamp(i);
          }
          else{
            turnOnLED();
          }
        }

        else{
          if(i<64){
            turnOffAllLamps();
          }
          else{
            turnOffLED();
          }
        }

        break;

      case 4:
        //brightness
        dimOutput(i);

        break;

      }
    }

    //DiagLEDControl(_lightControl[22]); //calling every end of loop using lamp 22
  }	
}

void dimOutput(int lampID){
  //based on the value of the brightness, we need to modulate the output pin to make a dimmer.
  //10 levels of brightness.
  //1 = 10%, 9 = 90%.
  int intensity;
  intensity = _brightness[lampID];
  int newValue;

  for(int i=1;i<100;i++){
    if(i % intensity ==0){
      //turn on 

      if(lampID <64){
        turnOnLamp(lampID);
      }
      else{
        //          Serial.println ("Turning On LED");
        //          PORTB ^=(1<<PB5);
        turnOnLED();
        //          _delay_ms(500); 
      }

    }
    else{
      //turn off
      if(lampID<64){
        turnOffAllLamps();
      }
      else{
        turnOffLED();
      }
    }
    delayMicroseconds(200);
  }



}



ISR(TIMER1_COMPA_vect)          // timer compare interrupt service routine
{  
  if(_timerControl>3){

    _timerControl = 0;

  }



  switch(_timerControl){

  case 0:
    _slowBlink=0x00;
    _fastBlink =0x00;

    break;

  case 1:
    _slowBlink = 0x00;
    _fastBlink=0x01;

    break;

  case 2:
    _slowBlink = 0x01;
    _fastBlink = 0x00;

    break;

  case 3:
    _slowBlink = 0x01;
    _fastBlink = 0x01;

    break;

  }



  _timerControl++;

}


//2 byte format. Byte 0 = 0 off,1 = on,2 = slow,3 = fast
//Byte 1 = Which Lamp we are controlling
//Byte 1 = 64, is the LED
//65 = controlling all lamps
void receiveEvent(int howMany)
{
  int lampCommand;

  digitalWrite(led, HIGH);   // turn the LED on (HIGH is the voltage level)

  lampCommand = Wire.read();
  _lastLamp = Wire.read();

  if(_lastLamp == 65 && lampCommand <4){
    //All Lamp Control
    SetLampAll(lampCommand);
  }
  
  if(_lastLamp < 65){
    if(lampCommand < 4){
      _lightControl[_lastLamp] = lampCommand;
    } else {
      //this is the brightness
      _lightControl[_lastLamp]=4;
      _brightness[_lastLamp] = lampCommand - 4;
    }
  }
  
  //  delay(1000);               // wait for a second
  /*  digitalWrite(led, LOW);    // turn the LED off by making the voltage LOW
   Serial.println("data received: ");
   Serial.print("number of bytes: ");
   Serial.println(howMany);
   Serial.print(lampCommand);
   Serial.print(":");
   Serial.println(_lastLamp);
   */
}

// callback for sending last receivedValue
void sendEvent(){
  Wire.write(_lastLamp);
}

/*
 J1 
 1 = Lamp En 1
 2 = Lamp En 2
 3 = Addr 4
 4 = Addr 3
 5 = Lamp En 3
 6 = Key
 7 = Lamp En 4
 8 = Addr 1
 9 = Addr 2
 
 Low PortB (Active High):
 0=lamp Address 0
 1=lamp Address 1
 2=lamp Address 2
 3=lamp Address 3
 
 Low PortC (Active low):
 0=str (Lamp En) 0
 1=str (Lamp En) 1
 2=str (Lamp En) 2
 3=str (Lamp En) 3
 
 LDU logic:
 4 Enables (Active Low) - port C
 4-Bit Data (Active High) - port B
 
 Example for Strobe 0, Lamp 5: 
 PortC = 0x#E
 PortB = 0x#5
 
 Valid Strobe values (0-3)= E (1110) ,D (1101), B (1011), 7 (0111)
 */
void turnOnLamp(int lampID){
  unsigned char newValue;

  //0xE0
  newValue =  PORTC | 0x0f; //reset low byte to all ####1111
  newValue = newValue & ((_ldu_map[lampID]>>4) | 0xf0); //example for 0xE0, this would be 0xfe AND-ing with. so new value should be ####1110
  PORTC = newValue;

  newValue = PORTB | 0x0f;  //reset low byte to all ####1111
  newValue = newValue & (_ldu_map[lampID] | 0xf0); //example for 0xE0, this would be 0xf0 AND-ing with. so new value would be ####0000
  PORTB = newValue;
}

void turnOnLED(){
  //          Serial.println ("Turning On LED");
  //          PORTB ^=(1<<PB5);
  PORTB = PORTB | 0x20; //turn on PB5 = LED 
  //          _delay_ms(500); 

}
void turnOffLED(){
  PORTB = PORTB & 0xdf; //turn off PB5 = LED   
}

void turnOffAllLamps(){
  PORTC |= 0x0f; //turn off.   
}


